package win32file

/*
   #include <windows.h>
   #include <stdlib.h>
   #include <string.h>

   static inline HANDLE CreateFileWin32(char* lpFileName, DWORD dwDesiredAccess, DWORD dwCreationDisposition, DWORD dwFlagsAndAttributes) {
       return CreateFile((LPCTSTR)(lpFileName), dwDesiredAccess, 0, NULL, dwCreationDisposition, dwFlagsAndAttributes, NULL);
   }
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

/* Wine 上で os パッケージを使ってファイルを書こうとすると「権限がない」というエラーが起こるので直接 Win32 API を叩くパッケージです。
 * 読み込みのフラグなども定義されていますが、今のところ上書きの書き込みのみ対応です。
 * というかぶっちゃけ log パッケージが使えればよかったので、そのあたりのことしかしてないです。
 */

type File struct {
	handle  C.HANDLE
	onWrite func([]byte) ([]byte, error) // Write()時に呼ばれるフックです。
}

const (
	O_RDONLY = 0x00001
	O_WRONLY = 0x00010
	O_RDWR   = O_RDONLY | O_WRONLY
)

func OpenFile(fileName string, flag int) (*File, error) {
	var dwDesiredAccess C.DWORD = 0
	lpFileName := C.CString(fileName)
	defer C.free((unsafe.Pointer)(lpFileName))
	if flag&O_RDONLY != 0 {
		dwDesiredAccess |= C.GENERIC_READ
	}
	if flag&O_WRONLY != 0 {
		dwDesiredAccess |= C.GENERIC_WRITE
	}
	hFile := C.CreateFileWin32(lpFileName, dwDesiredAccess, C.OPEN_ALWAYS, C.FILE_ATTRIBUTE_NORMAL)
	if hFile == nil {
		return nil, errors.New("CreateFile() failed: INVALID_HANDLE_VALUE")
	}
	return &File{handle: hFile}, nil
}

func (h *File) Close() {
	C.CloseHandle((*h).handle)
}

// io.Writer Interface
func (h *File) Write(p []byte) (int, error) {
	var err error
	var buf []byte
	var dwWriteSize C.DWORD
	if h.onWrite != nil {
		buf, err = h.onWrite(p)
	} else {
		buf = p
	}
	if err != nil {
		fmt.Printf("[error] onWrite() failed: %s\n", err.Error())
		return 0, err
	}
	lpBuffer := (C.LPVOID)(C.CBytes(buf))
	defer C.free((unsafe.Pointer)(lpBuffer))
	lpBufferLen := len(buf)
	ok := C.WriteFile((*h).handle, (C.LPCVOID)(lpBuffer), (C.DWORD)(lpBufferLen), &dwWriteSize, nil)
	if ok == C.FALSE {
		fmt.Printf("[error] C.WriteFile() failed: %d\n", C.GetLastError())
		return (int)(dwWriteSize), fmt.Errorf("%d", C.GetLastError())
	}
	return (int)(dwWriteSize), nil
}

// フックを設定するための関数です。
func (h *File) OnWrite(f func([]byte) ([]byte, error)) {
	h.onWrite = f
}
