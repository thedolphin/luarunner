// luarunner — a package for interacting with LuaJIT.
package luarunner

/*
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
	"runtime"
	"strconv"
	"time"
	"unsafe"
)

// LuaRunner represents Lua VM
type LuaRunner struct {
	L *C.lua_State
}

// New initializes and returns a new Lua VM
func New() (*LuaRunner, error) {
	L := C.luaL_newstate()
	if L == nil {
		return nil, errors.New("failed to initialize Lua VM")
	}

	C.luaL_openlibs(L)

	C.lua_init_extensions(L)

	return &LuaRunner{L}, nil
}

// CheckFunction checks for the existence of a global function in the current Lua VM
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

// Load loads a Lua script and pushes the code onto the top of the stack.
// To execute it, you need to call Run().
// code - Lua code
func (lua *LuaRunner) Load(code string) error {
	cCode := C.CString(code)
	defer C.free(unsafe.Pointer(cCode))

	res := int(C.luaL_loadstring(lua.L, cCode))
	if res != 0 {
		errorStr, err := lua.Pop()
		if err != nil {
			return fmt.Errorf("script execution error, failed to retrieve error description %w", err)
		}
		return fmt.Errorf("script execution error: %v", errorStr)
	}
	return nil
}

// Load loads a Lua script from a file and pushes the code onto the top of the stack.
// To execute it, you need to call Run().
// name - path to the file
func (lua *LuaRunner) LoadFile(name string) error {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	res := int(C.luaL_loadfilex(lua.L, cName, nil))
	if res != 0 {
		errorStr, err := lua.Pop()
		if err != nil {
			return fmt.Errorf("script execution error, failed to retrieve error description %w", err)
		}
		return fmt.Errorf("script execution error: %v", errorStr)
	}

	return nil
}

// Close releases all resources associated with the Lua VM
func (lua *LuaRunner) Close() {
	C.lua_close(lua.L)
}

// Push pushes a value onto the Lua VM stack.
//
// string and []byte are converted to a string (LUA_TSTRING);
// int, int32, int64, float32, float64 are converted to a number (LUA_TNUMBER),
// beware of truncating int64 to 53 bits;
// bool is converted to a boolean (LUA_TBOOLEAN);
// uintptr is converted to LUA_TLIGHTUSERDATA;
// nil is converted to LUA_TLIGHTUSERDATA(NULL) if it is a table value, otherwise to nil (LUA_TNIL).
//
// map[string]any is converted to a table (LUA_TTABLE), and it's the only map type
// that is efficiently passed to Lua. Other map types will use reflection.
// On one hand, this allows using arbitrary key and value types supported by Lua;
// on the other hand, reflection is less efficient. Another limitation is that
// this package does not allow retrieving a Lua table with key types other than string or number.
//
// Using reflection: pointers are dereferenced; slices and arrays are passed as tables
// with integer keys; structs and maps with arbitrary key/value types are passed as tables.
//
// If an unsupported data type is encountered, nil or LUA_TLIGHTUSERDATA(NULL) is pushed onto the stack,
// depending on whether it's a simple scalar or a table element.
func (lua *LuaRunner) Push(valueAny any) {
	lua.push(valueAny, false)
}

func (lua *LuaRunner) push(valueAny any, isTableValue bool) {
	switch value := valueAny.(type) {
	case nil:
		if isTableValue {
			// table value cannot be nil
			C.lua_pushlightuserdata(lua.L, unsafe.Pointer(nil))
		} else {
			C.lua_pushnil(lua.L)
		}
	case string:
		lua.pushString(value)
	case []byte:
		lua.pushBytes(value)
	case int:
		C.lua_pushinteger(lua.L, C.lua_Integer(value))
	case int32:
		C.lua_pushinteger(lua.L, C.lua_Integer(value))
	case int64:
		C.lua_pushinteger(lua.L, C.lua_Integer(value))
	case time.Time: // Lua function os.Date operates on the unix epoch
		C.lua_pushinteger(lua.L, C.lua_Integer(value.Unix()))
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
	cStr := (*C.char)(unsafe.Pointer(unsafe.StringData(str)))
	C.lua_pushlstring(lua.L, cStr, C.size_t(len(str)))
	runtime.KeepAlive(str)
}

func (lua *LuaRunner) pushBytes(data []byte) {
	cData := (*C.char)(unsafe.Pointer(unsafe.SliceData(data)))
	C.lua_pushlstring(lua.L, cData, C.size_t(len(data)))
	runtime.KeepAlive(data)
}

// Pop pops a value from Lua stack
func (lua *LuaRunner) Pop() (any, error) {
	defer C.lua_settop(lua.L, -2)
	return lua.Get(-1)
}

// Get gets a value in Lua stack by index
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
		// cjson and yyjson return LUA_TLIGHTUSERDATA(NULL) when decoding null in JSON
		// For comparison, use cjson.null or yyjson.null
		return nil, nil
	case C.LUA_TTABLE:
		result := make(map[string]any)
		C.lua_pushnil(lua.L)
		for C.lua_next(lua.L, -2) != 0 {
			keyAny, err := lua.Get(-2) // ключ
			if err != nil {
				return nil, fmt.Errorf("error retrieving table element key: %w", err)
			}
			val, err := lua.Pop() // значение
			if err != nil {
				return nil, fmt.Errorf("error retrieving table element value: %w", err)
			}
			switch key := keyAny.(type) {
			case string:
				result[key] = val
			case float64:
				result[strconv.Itoa(int(key))] = val
			default:
				return nil, errors.New("the key of a table element can only be a string or a number")
			}
		}
		return result, nil
	default:
		return nil, errors.New("the value on the stack can only be nil, string, number, boolean, table, or Light Userdata")
	}
}

// GetGlobal pushes the value of the global variable with the name 'name' onto the stack
func (lua *LuaRunner) GetGlobal(name string) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	C.lua_getfield(lua.L, C.LUA_GLOBALSINDEX, cName)
}

// SetGlobal pops a value from the stack and sets it as the global variable with the name 'name'
func (lua *LuaRunner) SetGlobal(name string) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	C.lua_setfield(lua.L, C.LUA_GLOBALSINDEX, cName)
}

// Call calls the function at the top of the stack
// To set the function, use GetGlobal
// in - number of input parameters
// out - number of return values, -1 for a variable number
func (lua *LuaRunner) Call(in, out int) error {
	if C.lua_pcall(lua.L, C.int(in), C.int(out), 0) != 0 {
		errorStr, err := lua.Pop()
		if err != nil {
			return fmt.Errorf("function call error, failed to retrieve error description %w", err)
		}
		return fmt.Errorf("function call error: %v", errorStr)
	}
	return nil
}

// Run calls the function at the top of the stack
// with no parameters and a variable number of return values
// To execute the script after Load and LoadFile
func (lua *LuaRunner) Run() error {
	return lua.Call(0, C.LUA_MULTRET)
}

// GetStackSize returns the current depth of the Lua VM stack
// It is used to determine the number of return values after a function call
// and for debugging memory leaks
func (lua *LuaRunner) GetStackSize() int {
	return int(C.lua_gettop(lua.L))
}

// AddPackagePath adds a pattern to the file search path for require (package.path)
func (lua *LuaRunner) AddPackagePath(pattern string) {
	lua.Load("package.path = package.path .. \";" + pattern + "\"")
	lua.Run()
}

// AddPackageCPath adds a pattern to the file search path for require (package.cpath)
func (lua *LuaRunner) AddPackageCPath(pattern string) {
	lua.Load("package.cpath = package.cpath .. \";" + pattern + "\"")
	lua.Run()
}

// StrictRead prevents reading uninitialized variables,
// protecting against typos.
func (lua *LuaRunner) StrictRead() {
	lua.Load(`setmetatable(_G, { __index = function (_, key) error("Attempt to read an uninitialized variable: " .. tostring(key), 2) end })`)
	lua.Run()
}

// StrictWrite prevents writing to undeclared global variables;
// It is useful to call it after initializing the script.
func (lua *LuaRunner) StrictWrite() {
	lua.Load(`setmetatable(_G, { __newindex = function (_, key, _) error("Attempt to write to undeclared variable: " .. tostring(key), 2) end })`)
	lua.Run()
}
