# Go wrapper for LuaJIT with built-in extensions

## Lua extensions

### `hash` library

1. `hash.postgres` - `pg_hash_bytes` from PostgreSQL sources
1. `hash.murmur3` - `MurmurHash3_x86_32` by Austin Appleby
1. `hash.crc32` - from libiberty

### `cjson` library

Original library, [documentation](https://kyne.au/~mark/software/lua-cjson-manual.html)

1. `cjson`
1. `cjson_safe`

### `yyjson`

A fast and non-destructive Lua library for working with JSON.
The JSON document is stored in internal yyjson structures and is not marshalled or unmarshalled to Lua values.
Creating JSON arrays is not supported.

1. `yyjson.load` - parses JSON into a read-only object
1. `yyjson.load_mut` - parses JSON into a mutable object
1. `yyjson.new` - creates a new mutable JSON object
1. `yyjson.null` - null constant
1. `tostring(v)` - serializes a mutable JSON object to a string

#### examples

```lua
j = yyjson.new() -- create an empty JSON object
j = yyjson.load(json_string) -- load JSON for read-only access (faster)
j = yyjson.load_mut(json_string) -- load JSON for modification

-- access fields
print(j.int32)
print(j.object.nested_object.inner_float)

-- modify fields
j.uuid = nil -- delete
j.boolean = yyjson.null -- overwrite with null
j.newobject = {test = 1, test1 = {test2 = "test"}}
j.array[4] = 8
j.copy = j.object -- copy subtrees

-- serialize to json
tostring(j) -- full document
tostring(j.object) -- document part
```
