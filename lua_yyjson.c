/*
 * Copyright (c) 2025 Alexander Rumyantsev
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

#include <lua.h>
#include <lualib.h>
#include <lauxlib.h>

#include <yyjson.h>

typedef struct yyjson_node_struct {
    yyjson_doc *doc;
    yyjson_val *root;
    int is_root;
} yyjson_node;

typedef struct yyjson_mut_node_struct {
    yyjson_mut_doc *mut_doc;
    yyjson_mut_val *mut_root;
    int is_root;
} yyjson_mut_node;

int lua_yyjson_new(lua_State *L) {

    yyjson_mut_node *node = lua_newuserdata(L, sizeof(yyjson_mut_node));
    memset(node, 0, sizeof(yyjson_mut_node));

    node->is_root = 1;
    node->mut_doc = yyjson_mut_doc_new(NULL);
    node->mut_root = yyjson_mut_obj(node->mut_doc);
    yyjson_mut_doc_set_root(node->mut_doc, node->mut_root);

    luaL_getmetatable(L, "yyjson_mut_mt");
    lua_setmetatable(L, -2);

    return 1;
}

int lua_yyjson_load(lua_State *L) {

	size_t jsonlen;
	const char *json = luaL_checklstring(L, -1, &jsonlen);

    yyjson_node *node = lua_newuserdata(L, sizeof(yyjson_node));
    memset(node, 0, sizeof(yyjson_node));

    yyjson_read_err err;
    node->doc = yyjson_read_opts((char *)json, jsonlen, 0, NULL, &err);
    if (node->doc == NULL)
        return luaL_error(L, "error parsing json at %d: %s", err.pos, err.msg);

    node->root = yyjson_doc_get_root(node->doc);

    node->is_root = 1;

    luaL_getmetatable(L, "yyjson_mt");
    lua_setmetatable(L, -2);

    return 1;
}

int lua_yyjson_load_mut(lua_State *L) {

	size_t jsonlen;
	const char *json = luaL_checklstring(L, -1, &jsonlen);

    yyjson_mut_node *node = lua_newuserdata(L, sizeof(yyjson_mut_node));
    memset(node, 0, sizeof(yyjson_mut_node));

    yyjson_read_err err;
    yyjson_doc *doc = yyjson_read_opts((char *)json, jsonlen, 0, NULL, &err);
    if (doc == NULL)
        return luaL_error(L, "error parsing json at %d: %s", err.pos, err.msg);

    node->mut_doc = yyjson_doc_mut_copy(doc, NULL);
    node->mut_root = yyjson_mut_doc_get_root(node->mut_doc);
    node->is_root = 1;

    yyjson_doc_free(doc);

    luaL_getmetatable(L, "yyjson_mut_mt");
    lua_setmetatable(L, -2);

    return 1;
}

int lua_yyjson_index(lua_State *L) {

    yyjson_node *node, *newnode;
    yyjson_val *val;

    node = (yyjson_node *)luaL_checkudata(L, 1, "yyjson_mt");

    yyjson_type node_type = yyjson_get_type(node->root);

    if (node_type == YYJSON_TYPE_OBJ) {
        val = yyjson_obj_get(node->root, luaL_checkstring(L, 2));
    } else if (node_type == YYJSON_TYPE_ARR) {
        val = yyjson_arr_get(node->root, luaL_checkint(L, 2));
    } else {
        return luaL_error(L, "attempt to index scalar value");
    }

    yyjson_type value_type = yyjson_get_type(val);

    switch (value_type) {
    case YYJSON_TYPE_NONE:
        lua_pushnil(L);
        break;
    case YYJSON_TYPE_NULL:
        lua_pushlightuserdata(L, NULL);
        break;
    case YYJSON_TYPE_STR:
        lua_pushstring(L, yyjson_get_str(val));
        break;
    case YYJSON_TYPE_BOOL:
        lua_pushboolean(L, yyjson_get_bool(val));
        break;
    case YYJSON_TYPE_NUM:
        lua_pushnumber(L, yyjson_get_num(val));
        break;
    case YYJSON_TYPE_ARR:
    case YYJSON_TYPE_OBJ:
        newnode = lua_newuserdata(L, sizeof(yyjson_node));
        memset(newnode, 0, sizeof(yyjson_node));
        newnode->doc = node->doc;
        newnode->root = val;
        luaL_getmetatable(L, "yyjson_mt");
        lua_setmetatable(L, -2);
    }

    return 1;
}

int lua_yyjson_index_mut(lua_State *L) {

    yyjson_mut_node *node, *newnode;
    yyjson_mut_val *val;

    node = (yyjson_mut_node *)luaL_checkudata(L, 1, "yyjson_mut_mt");

    yyjson_type node_type = yyjson_mut_get_type(node->mut_root);

    if (node_type == YYJSON_TYPE_OBJ) {
        val = yyjson_mut_obj_get(node->mut_root, luaL_checkstring(L, 2));
    } else if (node_type == YYJSON_TYPE_ARR) {
        val = yyjson_mut_arr_get(node->mut_root, luaL_checkint(L, 2));
    } else {
        return luaL_error(L, "attempt to index scalar value");
    }

    yyjson_type value_type = yyjson_mut_get_type(val);

    switch (value_type) {
    case YYJSON_TYPE_NONE:
        lua_pushnil(L);
        break;
    case YYJSON_TYPE_NULL:
        lua_pushlightuserdata(L, NULL);
        break;
    case YYJSON_TYPE_STR:
        lua_pushstring(L, yyjson_mut_get_str(val));
        break;
    case YYJSON_TYPE_BOOL:
        lua_pushboolean(L, yyjson_mut_get_bool(val));
        break;
    case YYJSON_TYPE_NUM:
        lua_pushnumber(L, yyjson_mut_get_num(val));
        break;
    case YYJSON_TYPE_ARR:
    case YYJSON_TYPE_OBJ:
        newnode = lua_newuserdata(L, sizeof(yyjson_node));
        memset(newnode, 0, sizeof(yyjson_node));
        newnode->mut_doc = node->mut_doc;
        newnode->mut_root = val;
        luaL_getmetatable(L, "yyjson_mut_mt");
        lua_setmetatable(L, -2);
    }

    return 1;
}

int lua_yyjson_newindex(lua_State *L) {

    return luaL_error(L, "attempt to write to readonly object");
}

yyjson_mut_val *lua_yyjson_val(lua_State *L, int idx, yyjson_mut_node *dst_node) {

    const char *str;
    size_t strlen;
    double luanum;
    yyjson_mut_val *newval;
    yyjson_mut_node *src_node;

    int value_type = lua_type(L, idx);

    switch (value_type) {
    case LUA_TNIL:
        return NULL;

    case LUA_TBOOLEAN:
        return yyjson_mut_bool(dst_node->mut_doc, lua_toboolean(L, idx));

    case LUA_TNUMBER:
        luanum = lua_tonumber(L, idx);
        if (luanum == (int64_t)luanum)
            return yyjson_mut_int(dst_node->mut_doc, (int64_t)luanum);
        return yyjson_mut_real(dst_node->mut_doc, luanum);

    case LUA_TSTRING:
        str = lua_tolstring(L, idx, &strlen);
        return yyjson_mut_strncpy(dst_node->mut_doc, str, strlen);

    case LUA_TTABLE:
        newval = yyjson_mut_obj(dst_node->mut_doc);
        lua_pushnil(L);
        while(lua_next(L, -2) != 0) {
            yyjson_mut_obj_add(
                newval,
                lua_yyjson_val(L, -2, dst_node),
                lua_yyjson_val(L, -1, dst_node));
            lua_pop(L, 1);
        }
        return newval;

    case LUA_TUSERDATA:
        src_node = (yyjson_mut_node *)luaL_checkudata(L, idx, "yyjson_mut_mt");
        return yyjson_mut_val_mut_copy(dst_node->mut_doc, src_node->mut_root);

    case LUA_TLIGHTUSERDATA:
        if (lua_touserdata(L, 3) == NULL) {
            return yyjson_mut_null(dst_node->mut_doc);
        }
    default:
        luaL_error(L, "invalid value");
    }

    return NULL;
}

int lua_yyjson_newindex_mut(lua_State *L) {

    yyjson_mut_node *node = (yyjson_mut_node *)luaL_checkudata(L, 1, "yyjson_mut_mt");
    yyjson_type node_type = yyjson_mut_get_type(node->mut_root);
    yyjson_mut_val *val, *newval;

    const char * key;
    size_t keylen;

    newval = lua_yyjson_val(L, 3, node);

    if (newval)
        if (node_type == YYJSON_TYPE_OBJ) {
            key = luaL_checklstring(L, 2, &keylen);
            yyjson_mut_obj_put(
                node->mut_root,
                yyjson_mut_strncpy(node->mut_doc, key, keylen),
                newval);
        } else if (node_type == YYJSON_TYPE_ARR) {
            keylen = luaL_checkint(L, 2);
            yyjson_mut_arr_remove(node->mut_root, keylen);
            yyjson_mut_arr_insert(node->mut_root, newval, keylen);
        } else {
            return luaL_error(L, "attempt to index scalar value");
        }
    else
        if (node_type == YYJSON_TYPE_OBJ) {
            key = luaL_checklstring(L, 2, &keylen);
            yyjson_mut_obj_remove_keyn(node->mut_root, key, keylen);
        } else if (node_type == YYJSON_TYPE_ARR) {
            yyjson_mut_arr_remove(node->mut_root, luaL_checkint(L, 2));
        } else {
            return luaL_error(L, "attempt to index scalar value");
        }
        
    return 0;
}

int lua_yyjson_write_mut(lua_State *L) {
    yyjson_mut_node *node = (yyjson_mut_node *)luaL_checkudata(L, 1, "yyjson_mut_mt");
    size_t jsonlen;
    char* json = yyjson_mut_val_write(node->mut_root, 0, &jsonlen);
    lua_pushlstring(L, json, jsonlen);
    free(json);
    return 1;
}

int lua_yyjson_gc(lua_State *L) {

    yyjson_node *node = (yyjson_node *)luaL_checkudata(L, 1, "yyjson_mt");
    if (node->is_root) {
        yyjson_doc_free(node->doc);
    }

    return 0;
}

int lua_yyjson_gc_mut(lua_State *L) {

    yyjson_mut_node *node = (yyjson_mut_node *)luaL_checkudata(L, 1, "yyjson_mut_mt");
    if (node->is_root) {
        yyjson_mut_doc_free(node->mut_doc);
    }

    return 0;
}

static const luaL_Reg yyjson_funcs[] = {
    {"new", lua_yyjson_new},
    {"load", lua_yyjson_load},
    {"load_mut", lua_yyjson_load_mut},
	{NULL, NULL}
};

int luaopen_yyjson(lua_State *L) {

    luaL_newmetatable(L, "yyjson_mt");
    lua_pushcfunction(L, lua_yyjson_index);
    lua_setfield(L, -2, "__index");
    lua_pushcfunction(L, lua_yyjson_newindex);
    lua_setfield(L, -2, "__newindex");
    lua_pushcfunction(L, lua_yyjson_gc);
    lua_setfield(L, -2, "__gc");
    lua_pop(L, 1);

    luaL_newmetatable(L, "yyjson_mut_mt");
    lua_pushcfunction(L, lua_yyjson_index_mut);
    lua_setfield(L, -2, "__index");
    lua_pushcfunction(L, lua_yyjson_newindex_mut);
    lua_setfield(L, -2, "__newindex");
    lua_pushcfunction(L, lua_yyjson_gc_mut);
    lua_setfield(L, -2, "__gc");
    lua_pushcfunction(L, lua_yyjson_write_mut);
    lua_setfield(L, -2, "__tostring");
    lua_pop(L, 1);

    luaL_newlib(L, yyjson_funcs);

    lua_pushlightuserdata(L, NULL);
    lua_setfield(L, -2, "null");

    return 1;
}
