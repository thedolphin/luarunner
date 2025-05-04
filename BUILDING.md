# Building

You should define CGO_CFLAGS and CGO_LDFLAGS to appropriate location of headers and libraries of luajit and yyjson

## Dynamic build on Mac with Macports

```
export CGO_CFLAGS='-I/opt/local/include -I/opt/local/include/luajit-2.1'
export CGO_LDFLAGS='-L/opt/local/lib -lluajit-5.1 -lyyjson'
```

## Static build on Debian amd64
```
export CGO_CFLAGS='-I/usr/include -I/usr/include/luajit-2.1'
export CGO_LDFLAGS='/usr/lib/x86_64-linux-gnu/libluajit-5.1.a /usr/lib/x86_64-linux-gnu/libyyjson.a -lm'
```

## VSCode `settings.json` for Mac with Macports
```
{
    "gopls": {
        "analyses": {
            "unsafeptr": false
        }
    },
    "go.toolsEnvVars": {
        "CGO_CFLAGS": "-I/opt/local/include -I/opt/local/include/luajit-2.1",
        "CGO_LDFLAGS": "-L/opt/local/lib -lluajit-5.1 -lyyjson"
    }
}
```
