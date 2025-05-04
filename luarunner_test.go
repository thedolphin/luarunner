package luarunner_test

import (
	"testing"

	"github.com/thedolphin/luarunner"
)

func TestLuaRunner(t *testing.T) {

	tests := []struct {
		name string
		code string
	}{
		{"printTable", `
function printTable(t)
    for key, value in pairs(t) do
        if type(value) == "table" then
            print(key .. " = {")
            printTable(value)
            print("}")
        else
            print(key .. " = " .. tostring(value))
        end
    end
end`},

		{"hash.postgres", `
print(hash.postgres("test_string"))
`},

		{"cjson.encode", `
print(cjson.encode({intfield = 1, floatfield = 0.1, stringfield = "test"}))
`},

		{"cjson.decode", `
local json_string = [[
{
  "string": "Hello, World!",
  "number": 12345,
  "float": 123.45,
  "boolean": true,
  "null_value": null,
  "array": [1, 2, 3, 4, 5],
  "object": {
    "nested_string": "Nested string value",
    "nested_number": 42,
    "nested_boolean": false,
    "nested_array": [10, 20, 30],
    "nested_object": {
      "inner_string": "Inner object string",
      "inner_float": 3.14159
    }
  },
  "date": "2025-04-28T00:00:00Z",
  "uuid": "f47ac10b-58cc-4372-a567-0e02b2c3d479"
}
]]
printTable(cjson.decode(json_string))
`},
		{"yyjson.decode", `
local json_string = [[
{
  "string": "Hello, World!",
  "number": 12345,
  "float": 123.45,
  "int64": 9223372036854775807,
  "uint64": 18446744073709551615,
  "int32": 2147483647,
  "unt32": 4294967295,
  "boolean": true,
  "null_value": null,
  "array": [1, 2, 3, 4, 5],
  "object": {
    "nested_string": "Nested string value",
    "nested_number": 42,
    "nested_boolean": false,
    "nested_array": [10, 20, 30],
    "nested_object": {
      "inner_string": "Inner object string",
      "inner_float": 3.14159
    }
  },
  "date": "2025-04-28T00:00:00Z",
  "uuid": "f47ac10b-58cc-4372-a567-0e02b2c3d479"
}
]]

print("=== [yyjson load and modify] =======================")

-- yyo = yyjson.load('{"a": }')

yyjsonobj = yyjson.load_mut(json_string)
-- print(yyjsonobj.int32)
-- print(yyjsonobj.object.nested_object.inner_float)
-- yyjsonobj.uuid = nil
-- yyjsonobj.boolean = yyjson.null
-- yyjsonobj.newtestval = {test = 1, test1 = {test2 = nil}}
-- yyjsonobj.array[4] = 8
yyjsonobj.copy = yyjsonobj.object
print(tostring(yyjsonobj))

-- print("=== [yyjson create] =======================")

-- local newo = yyjson.new()
-- newo.test = 1
-- newo.test2 = yyjson.null
-- newo.test3 = {"testval", "testval1"}
-- newo.test4 = {test5 = "testval2"}
-- print(tostring(newo))

print("=== [yyjson done] =======================")

`},
	}

	lua, err := luarunner.New()

	if err != nil {
		t.Error("Failed to initialize Lua VM: ", err)
		return
	}

	defer lua.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err = lua.Load(test.code); err != nil {
				t.Error("Could not load script: ", err)
				return
			}

			if err = lua.Run(); err != nil {
				t.Error("Could not run script: ", err)
				return
			}

			if size := lua.GetStackSize(); size != 0 {
				t.Error("Stach is not empty: ", size)
			}
		})
	}
}
