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
#include <WinBase.h>
*/
import "C"

import (
	"errors"
	"os"
)

// ComputerName returns the NetBIOS machine name. There are edge-cases for
// a seemingly wrong machine name to be returned. See the reference below
// for information on when this occurs.
//
// See also MSDN, ``GetComputerName function,''
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms724295(v=vs.85).aspx
func ComputerName() (string, error) {
	var numOfWC C.DWORD
	if ok := C.GetComputerNameW(nil, &numOfWC); ok == C.FALSE {
		wideStr := make([]C.wchar_t, numOfWC+1)
		if rc := C.GetComputerNameW((*C.WCHAR)(&wideStr[0]), &numOfWC); rc == C.TRUE {
			if utf8Str, err := wideToMB(C.CP_UTF8, wideStr); err == nil {
				return utf8Str, nil
			}
		}
	}

	return "", errors.New("ComputerName: failed")
}

// SystemDirectory returns the machine's system path location; typically
// ``C:\WINDOWS\system32''.
func SystemDirectory() (string, error) {
	if numOfWC := C.GetSystemDirectoryW(nil, 0); numOfWC > 0 {
		wideStr := make([]C.wchar_t, numOfWC)
		if rc := C.GetSystemDirectoryW((*C.WCHAR)(&wideStr[0]), numOfWC); rc > 0 {
			if utf8Str, err := wideToMB(C.CP_UTF8, wideStr); err == nil {
				return utf8Str, nil
			}
		}
	}

	return "", errors.New("SystemDirectory: failed")
}

// HomeDirectory returns the current user's directory on the machine; typically
// a folder inside the ``C:\Users'' directory.
func HomeDirectory() (string, error) {
	if s, ok := os.LookupEnv("USERPROFILE"); ok {
		return s, nil
	}
	if s, ok := os.LookupEnv("HOMEDRIVE"); ok {
		if s1, ok1 := os.LookupEnv("HOMEPATH"); ok1 {
			return s + s1, nil
		}
	}
	return SystemDirectory()
}

// ConfigHomeDirectory returns the current user's application configuration
// directory on the user's roaming profile. All configuration file written
// are possibly synchronized between multiple machines the user may have
// access to.
func ConfigHomeDirectory() (string, error) {
	if s, ok := os.LookupEnv("APPDATA"); ok {
		return s, nil
	}
	return HomeDirectory()
}

// DataHomeDirectory returns the current user's application data configuration
// directory on the user's local, specific to the current machine, profile.
// All configuration data written are only stored on the current machine. For
// possibly synchronized configuration data, see ConfigHomeDirectory().
func DataHomeDirectory() (string, error) {
	if s, ok := os.LookupEnv("LOCALAPPDATA"); ok {
		return s, nil
	}
	return ConfigHomeDirectory()
}

// ConfigDirectory returns the running machine's application configuration
// and/or local data directory. Write access may require Administrator
// privileges.
func ConfigDirectory() (string, error) {
	if s, ok := os.LookupEnv("PROGRAMDATA"); ok {
		return s, nil
	}
	return SystemDirectory()
}
