/* This is composed stub header file for hashfn.c */

#include <stdint.h>
#include <stdlib.h>
#include <assert.h>
#include <string.h>

#define HAVE_INT64 1

typedef unsigned int uint32;

#ifndef HAVE_INT64
typedef long int int64;
#endif
#ifndef HAVE_UINT64
typedef unsigned long int uint64;
#endif

typedef size_t Size;

static inline uint32
pg_rotate_left32(uint32 word, int n)
{
	return (word << n) | (word >> (32 - n));
}

typedef unsigned char bool;

#define true			((bool) 1)
#define Assert(condition)	((void)true)
#define Min(x, y)		((x) < (y) ? (x) : (y))

uint32 hash_bytes(const unsigned char *k, int keylen);
