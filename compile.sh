#!/usr/bin/sh

### Create 32bit Windows DLL

    #################################################
    #
    echo 'Compile "shiori.go" and create "libshiori.a"'
    #
    #################################################
    
    CGO_ENABLED=1 GOOS="windows" GOARCH="386" CC="i686-w64-mingw32-gcc-win32" go build -buildmode=c-archive -o libshiori.a

RET="$?"

if [ $RET = 0 ]; then

    #################################################
    #
    echo 'Create "shiori.dll" from "libshiori.a" and "shiori.def"'
    #
    #################################################
    
    i686-w64-mingw32-gcc-win32 -shared -o shiori.dll shiori.def libshiori.a -Wl,--allow-multiple-definition -static -lstdc++ -lwinmm -lntdll -lws2_32

RET="$?"
fi

if [ $RET = 0 ]; then
    
    #################################################
    #
    echo 'Copy "shiori.dll" to "gohst/ghost/master/shiori.dll"'
    #
    #################################################
    
    cp shiori.dll gohst/ghost/master/shiori.dll

RET="$?"
fi

if [ $RET = 0 ]; then
    
    #################################################
    #
    echo 'Zip "gohst" to "gohst.zip", changing LF to CR+LF'
    #
    #################################################
    
    zip -r -q -l gohst.zip gohst

RET="$?"
fi

if [ $RET = 0 ] && [ -e "gohst.zip" ]; then
    
    #################################################
    #
    echo 'Create "ghost.nar" from "gohst.zip"'
    #
    #################################################
    
    mv gohst.zip gohst.nar
fi
