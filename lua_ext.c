#include <lua.h>
#include <lualib.h>
#include <lauxlib.h>

#include "hashfn.h"
#include "murmur3.h"

unsigned int xcrc32 (const unsigned char *buf, int len, unsigned int init);

int luaopen_cjson(lua_State *l);
int luaopen_cjson_safe(lua_State *l);
int luaopen_yyjson(lua_State *L);

static int lua_pg_hash_bytes(lua_State *L) {
	Size len;
	const unsigned char *input = (const unsigned char *)lua_tolstring(L, -1, &len);
	uint32 hash = hash_bytes(input, len);
	lua_pushinteger(L, hash);
	return 1;
}

static int lua_murmur3(lua_State *L) {
	size_t len;
	uint32_t hash;
	const void *input = (const void *)lua_tolstring(L, -1, &len);
	MurmurHash3_x86_32(input, len, 0, &hash);
	lua_pushinteger(L, hash);
	return 1;
}

static int lua_crc32(lua_State *L) {
	size_t len;
	unsigned int hash;
	const void *input = (const void *)lua_tolstring(L, -1, &len);
	hash = xcrc32(input, len, 0xffffffff);
	lua_pushinteger(L, hash);
	return 1;
}

static const luaL_Reg hashfuncs[] = {
	{"postgres", lua_pg_hash_bytes},
	{"murmur3", lua_murmur3},
	{"crc32", lua_crc32},
	{NULL, NULL}
};

void lua_init_extensions(lua_State *L) {

	lua_getglobal(L, "_G");

	luaL_newlib(L, hashfuncs);
	lua_setfield(L, -2, "hash");

	luaopen_cjson(L);
	lua_setfield(L, -2, "cjson");

	luaopen_cjson_safe(L);
	lua_setfield(L, -2, "cjson_safe");

	luaopen_yyjson(L);
	lua_setfield(L, -2, "yyjson");

	lua_settop(L, -2);
}
