// luarunner - пакет для взаимодействия с Lua
package luarunner

/*
#cgo darwin CFLAGS: -I/opt/local/include -I/opt/local/include/luajit-2.1
#cgo darwin LDFLAGS: -L/opt/local/lib -lluajit-5.1 -lyyjson

#cgo linux CFLAGS: -I/usr/include -I/usr/include/luajit-2.1
#cgo linux amd64 LDFLAGS: /usr/lib/x86_64-linux-gnu/libluajit-5.1.a /usr/lib/x86_64-linux-gnu/libyyjson.a -lm

#include <stdlib.h>

#include <lua.h>
#include <lualib.h>
#include <lauxlib.h>

int lua_init_extensions(lua_State *L);
*/
import "C"
import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"unsafe"
)

// LuaRunner - виртуальная машина Lua
type LuaRunner struct {
	L *C.lua_State
}

// New инициализирует и возвращает новую машину Lua
func New() (*LuaRunner, error) {
	L := C.luaL_newstate()
	if L == nil {
		return nil, errors.New("не удалось инициализировать Lua")
	}

	C.luaL_openlibs(L)

	C.lua_init_extensions(L)

	return &LuaRunner{L}, nil
}

// CheckFunction проверяет существование глобальной функции в текущей машине LUA
func (lua *LuaRunner) CheckFunction(funcName string) bool {
	lua.GetGlobal(funcName)
	v := C.lua_type(lua.L, -1)
	if v != C.LUA_TFUNCTION {
		lua.Close()
		return false
	}
	C.lua_settop(lua.L, -2)
	return true
}

// Load загружает скрипт Lua и кладет код на вершину стека
// Для исполнения нужно вызвать Run()
// code - код Lua
func (lua *LuaRunner) Load(code string) error {
	cCode := C.CString(code)
	defer C.free(unsafe.Pointer(cCode))

	res := int(C.luaL_loadstring(lua.L, cCode))
	if res != 0 {
		errorStr, err := lua.Pop()
		if err != nil {
			return fmt.Errorf("ошибка выполнения скрипта, ошибка получения описания ошибки %w", err)
		}
		return fmt.Errorf("ошибка выполнения скрипта: %v", errorStr)
	}
	return nil
}

// Load загружает скрипт Lua из файла и кладёт код на вершину стека
// Для исполнения нужно вызвать Run()
// name - путь к файлу
func (lua *LuaRunner) LoadFile(name string) error {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	res := int(C.luaL_loadfilex(lua.L, cName, nil))
	if res != 0 {
		errorStr, err := lua.Pop()
		if err != nil {
			return fmt.Errorf("ошибка загрузки скрипта, ошибка получения описания ошибки %w", err)
		}
		return fmt.Errorf("ошибка загрузки скрипта: %v", errorStr)
	}

	return nil
}

// Close освобождает все ресурсы связанные с машиной Lua
func (lua *LuaRunner) Close() {
	C.lua_close(lua.L)
}

// Push кладёт в стек машины Lua значение
//
// string и[]bytes преобразуется в строку (LUA_TSTRING);
// int, int32, int64, float32, float64 преобразуются в число (LUA_TNUMBER);
// bool преобразуется в булево (LUA_TBOOLEAN);
// uintptr преобразуется в LUA_TLIGHTUSERDATA;
// nil преобразуется в LUA_TLIGHTUSERDATA(NULL) если это значение таблицы, иначе nil (LUA_TNIL).
//
// map[string]any преобразуется в таблицу (LUA_TTABLE), и это единственный вид map,
// который будет эффективно передаваться в Lua, для других видов map будет использоваться
// рефлексия. С одной стороны это позволяет использовать любые типы ключей и значений,
// что поддерживается в Lua, с другой - рефлексия менее эффективна. Другим ограничением является
// то, что этот пакет не позволяет получить из Lua таблицу с типами ключей отличными от строки и числа.
//
// С помощью рефлексии: разыменовываются указатели; передаются срезы и массивы в виде таблицы
// с целочисленными ключами; передаются структуры и карты с произвольными типами ключей и значений.
//
// При получении неподдерживаемого типа данных в стек кладётся nil или LUA_TLIGHTUSERDATA(NULL)
func (lua *LuaRunner) Push(valueAny any) {
	lua.push(valueAny, false)
}

func (lua *LuaRunner) push(valueAny any, isTableValue bool) {
	switch value := valueAny.(type) {
	case nil:
		if isTableValue {
			// значение элемента таблицы не может быть nil
			C.lua_pushlightuserdata(lua.L, unsafe.Pointer(nil))
		} else {
			C.lua_pushnil(lua.L)
		}
	case string:
		lua.pushString(value)
	case []byte:
		lua.pushString(string(value))
	case int:
		C.lua_pushinteger(lua.L, C.lua_Integer(value))
	case int32:
		C.lua_pushinteger(lua.L, C.lua_Integer(value))
	case int64:
		C.lua_pushinteger(lua.L, C.lua_Integer(value))
	case float32:
		C.lua_pushnumber(lua.L, C.lua_Number(value))
	case float64:
		C.lua_pushnumber(lua.L, C.lua_Number(value))
	case uintptr:
		C.lua_pushlightuserdata(lua.L, unsafe.Pointer(value))
	case bool:
		if value {
			C.lua_pushboolean(lua.L, 1)
		} else {
			C.lua_pushboolean(lua.L, 0)
		}
	case map[string]any:
		C.lua_createtable(lua.L, 0, 0)
		for key, fieldValue := range value {
			lua.pushString(key)
			lua.push(fieldValue, true)
			C.lua_settable(lua.L, -3)
		}
	default:
		lua.pushReflect(value, isTableValue)
	}
}

// pushReflect кладёт некоторые типы значений в стек Lua через рефлексию
func (lua *LuaRunner) pushReflect(valueAny any, isTableValue bool) {
	reflectValue := reflect.ValueOf(valueAny)

	switch reflectValue.Kind() {
	case reflect.Pointer:
		lua.push(reflectValue.Elem().Interface(), isTableValue)
	case reflect.Slice, reflect.Array:
		C.lua_createtable(lua.L, 0, 0)
		for i := range reflectValue.Len() {
			lua.push(i, false)
			lua.push(reflectValue.Index(i).Interface(), true)
			C.lua_settable(lua.L, -3)
		}
	case reflect.Map:
		C.lua_createtable(lua.L, 0, 0)
		for _, reflectKey := range reflectValue.MapKeys() {
			lua.push(reflectKey.Interface(), false)
			lua.push(reflectValue.MapIndex(reflectKey).Interface(), true)
			C.lua_settable(lua.L, -3)
		}
	case reflect.Struct:
		C.lua_createtable(lua.L, 0, 0)
		reflectType := reflect.TypeOf(valueAny)
		for i := range reflectValue.NumField() {
			lua.push(reflectType.Field(i).Name, false)
			lua.push(reflectValue.Field(i).Interface(), true)
			C.lua_settable(lua.L, -3)
		}
	default:
		lua.push(nil, isTableValue)
	}
}

func (lua *LuaRunner) pushString(str string) {
	cStr := C.CString(str)
	defer C.free(unsafe.Pointer(cStr))

	C.lua_pushlstring(lua.L, cStr, C.size_t(len(str)))
}

// Pop вынимает значение из стека машины Lua
func (lua *LuaRunner) Pop() (any, error) {
	defer C.lua_settop(lua.L, -2)
	return lua.Get(-1)
}

// Get получает значение из стека машины Lua на глубине i без вынимания
func (lua *LuaRunner) Get(i int) (any, error) {

	switch C.lua_type(lua.L, C.int(i)) {
	case C.LUA_TNIL:
		return nil, nil
	case C.LUA_TSTRING:
		return C.GoString(C.lua_tolstring(lua.L, C.int(i), nil)), nil
	case C.LUA_TNUMBER:
		return float64(C.lua_tonumberx(lua.L, C.int(i), nil)), nil
	case C.LUA_TBOOLEAN:
		return C.lua_toboolean(lua.L, C.int(i)) != 0, nil
	case C.LUA_TLIGHTUSERDATA:
		if val := C.lua_touserdata(lua.L, C.int(i)); val != nil {
			return uintptr(val), nil
		}
		// cjson возвращает LUA_TLIGHTUSERDATA(NULL) при декодировании null в json
		// для сравнивания есть константа cjson.null
		return nil, nil
	case C.LUA_TTABLE:
		result := make(map[string]any)
		C.lua_pushnil(lua.L)
		for C.lua_next(lua.L, -2) != 0 {
			keyAny, err := lua.Get(-2) // ключ
			if err != nil {
				return nil, fmt.Errorf("ошибка выбора ключа элемента таблицы: %w", err)
			}
			val, err := lua.Pop() // значение
			if err != nil {
				return nil, fmt.Errorf("ошибка выбора значения элемента таблицы: %w", err)
			}
			switch key := keyAny.(type) {
			case string:
				result[key] = val
			case float64:
				result[strconv.Itoa(int(key))] = val
			default:
				return nil, errors.New("ключом элемента таблицы может быть только строка или число")
			}
		}
		return result, nil
	default:
		return nil, errors.New("значение в стеке может быть только nil, строка, число, булево, таблица или Light Userdata")
	}
}

// GetGlobal кладёт в стек значение глобальной переменной с именем name
func (lua *LuaRunner) GetGlobal(name string) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	C.lua_getfield(lua.L, C.LUA_GLOBALSINDEX, cName)
}

// SetGlobal вынимает из стека значение и устанавливает
// в качестве глобальной переменной с именем name
func (lua *LuaRunner) SetGlobal(name string) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	C.lua_setfield(lua.L, C.LUA_GLOBALSINDEX, cName)
}

// Call вызывает функцию, лежащую на вершине стека
// Для задания функции надо использовать GetGlobal
// in - количество параметров
// out - количество возвращаемых значений, -1 для произвольного числа
func (lua *LuaRunner) Call(in, out int) error {
	if C.lua_pcall(lua.L, C.int(in), C.int(out), 0) != 0 {
		errorStr, err := lua.Pop()
		if err != nil {
			return fmt.Errorf("ошибка вызова функции, ошибка получения описания ошибки %w", err)
		}
		return fmt.Errorf("ошибка вызова функции: %v", errorStr)
	}
	return nil
}

// Run вызывает функцию, лежащую на вершине стека
// без параметров и с произвольным количеством возвращаемых значений
// Для запуска скрипта после Load и LoadFile
func (lua *LuaRunner) Run() error {
	return lua.Call(0, C.LUA_MULTRET)
}

// GetStackSize возвращает текущую глубину стека машины Lua
// Нужно для определения количества возвращенных значений после вызова функции
// и для отладки утечек
func (lua *LuaRunner) GetStackSize() int {
	return int(C.lua_gettop(lua.L))
}

// AddPackagePath добавляет паттерн к пути поиска файлов при вызове require (package.path)
func (lua *LuaRunner) AddPackagePath(pattern string) {
	lua.Load("package.path = package.path .. \";" + pattern + "\"")
	lua.Run()
}

// AddPackageCPath добавляет паттерн к пути поиска файлов при вызове require (package.cpath)
func (lua *LuaRunner) AddPackageCPath(pattern string) {
	lua.Load("package.cpath = package.cpath .. \";" + pattern + "\"")
	lua.Run()
}

// StrictRead запрещает чтение неинициализированной переменной,
// защита от опечаток
func (lua *LuaRunner) StrictRead() {
	lua.Load(`setmetatable(_G, { __index = function (_, key) error("Attempt to read an uninitialized variable: " .. tostring(key), 2) end })`)
	lua.Run()
}

// StrictWrite запрещает запись в необъявленную переменную,
// заставляя явно объявлять переменную через local
func (lua *LuaRunner) StrictWrite() {
	lua.Load(`setmetatable(_G, { __newindex = function (_, key, _) error("Attempt to write to undeclared variable: " .. tostring(key), 2) end })`)
	lua.Run()
}
