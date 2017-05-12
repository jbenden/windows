/*
Copyright 2017 Joseph Benden <joe@benden.us>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package windows

/*
#cgo windows CFLAGS: -D_UNICODE -DUNICODE -DWIN32 -DWINVER=0x0600 -I/usr/local/w32api
#include <windows.h>
#include <Stringapiset.h>
#include <Winnls.h>
*/
import "C"

import (
	"errors"
	"unsafe"
)

// Possible Windows Code Pages
const (
	CP1252      int = 1252
	UnicodeFFFE     = 1201
	Macintosh       = 10000
	UTF32           = 12000
	UTF32BE         = 12001
	UsASCII         = 20127
	ISO88591        = 28591
	ISO88592        = 28592
	UTF7            = 65000
	UTF8            = 65001
)

// ErrInvalidEncoding is returned as a result of an invalid conversion.
var ErrInvalidEncoding = errors.New("windows: invalid string encoding")

// ErrInvalidNarrow is returned as a result of an invalid conversion to a narrow byte sequence.
var ErrInvalidNarrow = errors.New("windows: invalid narrow encoded string")

// ErrInvalidWide is returned as a result of an invalid conversion to a wide-character sequence.
var ErrInvalidWide = errors.New("windows: invalid wide-character encoded string")

// GetSystemCodePage returns Window's default system code page.
func GetSystemCodePage() int {
	var cpInfoEx C.CPINFOEX

	if ok := C.GetCPInfoEx(C.CP_ACP, 0, &cpInfoEx); ok == C.TRUE {
		return (int)(cpInfoEx.CodePage)
	}

	return 0
}

// wideToMB converts and wide-character sequence to a multi-byte character sequence.
func wideToMB(codePage C.UINT, wide []C.wchar_t) (string, error) {
	if numOfMB := C.WideCharToMultiByte(codePage, 0 /*C.WC_ERR_INVALID_CHARS*/, (*C.WCHAR)(&wide[0]), -1, nil, 0, nil, nil); numOfMB > 0 {
		mbStr := make([]C.char, numOfMB)
		if rc := C.WideCharToMultiByte(codePage, 0 /*C.WC_ERR_INVALID_CHARS*/, (*C.WCHAR)(&wide[0]), -1, (*C.CHAR)(&mbStr[0]), numOfMB, nil, nil); rc > 0 {
			ptr := (*C.char)(unsafe.Pointer(&mbStr[0])) // #nosec
			return C.GoString(ptr), nil
		}
	}

	return "", ErrInvalidNarrow
}

// mbToWide converts a multi-byte character sequence to wide-character sequence.
func mbToWide(codePage C.UINT, mb *C.char) ([]C.wchar_t, error) {
	if numOfWC := C.MultiByteToWideChar(codePage, C.MB_ERR_INVALID_CHARS, (*C.CHAR)(mb), -1, nil, 0); numOfWC > 0 {
		wideStr := make([]C.wchar_t, numOfWC)
		if rc := C.MultiByteToWideChar(codePage, C.MB_ERR_INVALID_CHARS, (*C.CHAR)(mb), -1, (*C.WCHAR)(&wideStr[0]), numOfWC); rc > 0 {
			for _, ch := range wideStr {
				if ch == 0xFFFD {
					return wideStr, ErrInvalidWide
				}
			}
			return wideStr, nil
		}
	}

	return nil, ErrInvalidWide
}

// SystemCodePageToUtf8 converts the given string from Window's system code page to an UTF-8 string.
func SystemCodePageToUtf8(s string) (string, error) {
	str := C.CString(s)
	defer C.free(unsafe.Pointer(str)) // #nosec

	if wcACPStr, err := mbToWide(C.CP_ACP, str); err == nil {
		if utf8Str, err := wideToMB(C.CP_UTF8, wcACPStr); err == nil {
			return utf8Str, nil
		}
	}

	return s, ErrInvalidEncoding
}

// Utf8ToSystemCodePage converts the given UTF-8 string to Window's system code page.
func Utf8ToSystemCodePage(s string) (string, error) {
	str := C.CString(s)
	defer C.free(unsafe.Pointer(str)) // #nosec

	if wcACPStr, err := mbToWide(C.CP_UTF8, str); err == nil {
		if utf8Str, err := wideToMB(C.CP_ACP, wcACPStr); err == nil {
			return utf8Str, nil
		}
	}

	return s, ErrInvalidEncoding
}
