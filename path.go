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

import (
	"bytes"
	"errors"
	"os"
	"syscall"
	"unicode/utf8"
)

// ErrInvalidDrive indicates an invalid character was used as a drive letter.
var ErrInvalidDrive = errors.New("path: invalid drive specified")

// PathImpl holds state between each of the functional calls returned by Path().
type PathImpl struct {
	node     string
	device   string
	name     string
	dirs     []string
	absolute bool
	unc      bool
	unicode  bool
	errs     []error
}

func isDriveLetter(c rune) (rune, error) {
	switch {
	case c >= 'a' && c <= 'z':
		return c - 32, nil
	case c >= 'A' && c <= 'Z':
		return c, nil
	default:
		return -1, errors.New("isDriveLetter: an invalid drive letter was found")
	}
}

// isPathNameLetter determines if the given rune is valid for a path or file name.
//
// See: https://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx
func isPathNameLetter(c rune) (rune, error) {
	switch c {
	case '<', '>', ':', '"', '/', '\\', '|', '?', '*':
		// reserved characters
		return -1, errors.New("isPathNameLetter: a reserved character is present")
	default:
		switch {
		case c == 0:
			// cannot be the NULL character
			return c, errors.New("isPathNameLetter: a NULL rune is present")
		case c >= 1 && c <= 31:
			// invalid except for alternate data streams
			return c, errors.New("isPathNameLetter: an invalid rune is present; unless a File Stream")
		default:
			// Valid for GENERAL Windows file naming rules, although FS may impose additional restrictions
			return c, nil
		}
	}
}

const (
	stateStart int = iota
	stateUNC
	stateDrive
	statePathComponent
)

const (
	substateStart int = iota
	substateUnicode
	substateUnicodeUNC
)

// newPathImpl parses and returns a new PathImpl from a given string.
//
// See Path() for more.
func newPathImpl(path string) *PathImpl {
	_path := &PathImpl{}

	runeArrayLen := utf8.RuneCountInString(path)
	runeArray := make([]rune, runeArrayLen)

	for index, runeValue := range path {
		runeArray[index] = runeValue
	}

	curIdx := 0
	curState := stateStart
	curSubState := substateStart
	var curStack []rune
loopStart:
	for curIdx < runeArrayLen {
		switch curState {
		case stateStart:
			if runeArray[curIdx] == '\\' && curIdx+1 < runeArrayLen && runeArray[curIdx+1] == '\\' {
				// UNC path
				curIdx++
				curState = stateUNC
			} else if runeArray[curIdx] == '\\' {
				// Abs path
				// BEGIN PathState
				_path.absolute = true
				curState = statePathComponent
			} else {
				curState = stateDrive
				goto loopStart
			}
		case stateDrive:
			// BEGIN DriveLetterState
			c, err := isDriveLetter(runeArray[curIdx])
			if err == nil {
				// Drive letter was found, but is the next rune a colon?
				if curIdx+1 < runeArrayLen && runeArray[curIdx+1] == ':' {
					// for sure a drive, start path parsing state
					_path.device = string(c)

					if curIdx+2 < runeArrayLen && runeArray[curIdx+2] == '\\' {
						_path.absolute = true
					}
					// else RELATIVE to current working directory, but on this OTHER drive!

					curIdx++
					curIdx++
					// was a drive, start path components
					curState = statePathComponent
					goto loopStart
				} else {
					// was a drive, start path components
					curState = statePathComponent
					goto loopStart
				}
			}
			// not a drive, but maybe a relative path?
			curState = statePathComponent
			goto loopStart
		case statePathComponent:
			// either have a path or a file name as the possibility...
			if runeArray[curIdx] == '\\' {
				if len(curStack) > 0 {
					// finished a component of the path, push and continue
					_path.dirs = append(_path.dirs, string(curStack))
					curStack = make([]rune, 0, 160)
				}
			} else {
				if _, err := isPathNameLetter(runeArray[curIdx]); err != nil {
					_path.errs = append(_path.errs, err)
				}
				curStack = append(curStack, runeArray[curIdx])
			}
		case stateUNC:
			if runeArray[curIdx] == '\\' {
				if len(curStack) > 0 {
					// finished a component of the node
					node := string(curStack)
					curStack = make([]rune, 0, 160)

					switch curSubState {
					case substateUnicode:
						if node == "UNC" {
							// UNICODE UNC path, but UNC share
							curIdx++
							curSubState = substateUnicodeUNC
							curState = stateUNC
							goto loopStart
						} else {
							curIdx -= len(node)
							curState = stateDrive
							goto loopStart
						}
					case substateStart:
						if node == "?" {
							// UNICODE UNC has been specified
							curIdx++
							curSubState = substateUnicode
							curState = stateUNC
							_path.unicode = true
							goto loopStart
						}
						fallthrough
					case substateUnicodeUNC:
						fallthrough
					default:
						// add component
						_path.node = node
						_path.unc = true
						curState = statePathComponent
					}
				}
			} else {
				if _, err := isPathNameLetter(runeArray[curIdx]); err != nil {
					_path.errs = append(_path.errs, err)
				}
				curStack = append(curStack, runeArray[curIdx])
			}
		}

		curIdx++
	}

	// If a last curStack is present, it is actually the filename being accessed or a final dir
	if len(curStack) > 0 {
		for _, c := range curStack {
			if _, err := isPathNameLetter(c); err != nil {
				_path.errs = append(_path.errs, err)
			}
		}
		_path.name = string(curStack)
	}

	if _path.unicode && len(path) > 32767 {
		_path.errs = append(_path.errs, errors.New("Path: the UNICODE path exceeds the maximum of 32,767 characters"))
	} else if !_path.unicode && len(path) > 255 {
		_path.errs = append(_path.errs, errors.New("Path: the path exceeds the maximum of 255 characters"))
	}

	return _path
}

// ToString returns a fully-qualified representation of the parsed Path.
func (p *PathImpl) ToString() string {
	var unc bytes.Buffer
	hasComponents := false

	if len(p.device) > 0 {
		unc.WriteString(p.device)
		unc.WriteString(":")
	}
	if len(p.node) > 0 {
		unc.WriteString("\\\\")
		unc.WriteString(p.node)
	}
	for _, path := range p.dirs {
		hasComponents = true
		unc.WriteString("\\")
		unc.WriteString(path)
	}
	if len(p.name) > 0 {
		hasComponents = true
		unc.WriteString("\\")
		unc.WriteString(p.name)
	}
	if !hasComponents {
		unc.WriteString("\\")
	}

	return unc.String()
}

// ToUnicodeUNC returns a fully-qualified UNICODE UNC representation of the parsed Path.
func (p *PathImpl) ToUnicodeUNC() string {
	var unc bytes.Buffer

	unc.WriteString("\\\\?\\")
	if len(p.device) > 0 {
		unc.WriteString(p.device)
		unc.WriteString(":\\")
	}
	if len(p.node) > 0 {
		unc.WriteString("UNC\\")
		unc.WriteString(p.node)
		unc.WriteString("\\")
	}
	for _, path := range p.dirs {
		unc.WriteString(path)
		unc.WriteString("\\")
	}
	unc.WriteString(p.name)

	return unc.String()
}

// IsDirectoryExists checks whether the Path refers to an existing directory.
func (p *PathImpl) IsDirectoryExists() bool {
	if fi, err := os.Stat(p.ToString()); err == nil {
		switch mode := fi.Mode(); {
		case mode.IsDir():
			return true
		}
	}
	return false
}

// IsAbsolute checks whether the Path refers to a non-relative location.
func (p *PathImpl) IsAbsolute() bool {
	return p.absolute
}

// IsRelative checks whether the Path refers to a non-absolute location.
func (p *PathImpl) IsRelative() bool {
	return !p.absolute
}

// IsRemote checks whether the Path refers to a location that is not on the current machine.
func (p *PathImpl) IsRemote() bool {
	return p.unc
}

// IsLocal checks whether the Path refers to a location on the current machine.
func (p *PathImpl) IsLocal() bool {
	return !p.unc
}

// MakeAbsolute checks whether the Path refers to a relative location on the
// current machine. If so, it non-destructively converts the relative location
// to an absolute one by querying the path through the operating system.
//
// Because MakeAbsolute is non-destructive, the returned pointer to PathImpl
// may NOT be the same as called with!
func (p *PathImpl) MakeAbsolute() *PathImpl {
	if !p.unc && !p.absolute {
		if newPath, err := syscall.FullPath(p.ToString()); err == nil {
			return Path(newPath)
		}
	}
	return p
}

// MakeDirectory checks whether the Path has a Name(). If so, the
// Name() is added to the set of Dirs() and is cleared.
func (p *PathImpl) MakeDirectory() *PathImpl {
	if len(p.name) > 0 {
		p.dirs = append(p.dirs, p.name)
		p.name = ""
	}
	return p
}

// Node returns the server name from a parsed UNC path.
func (p *PathImpl) Node() string {
	return p.node
}

// Device returns the drive letter from a parsed absolute path.
func (p *PathImpl) Device() string {
	return p.device
}

// Name returns the file name or last directory from a parsed path.
func (p *PathImpl) Name() string {
	return p.name
}

// Dirs returns an array of all parsed leading directories.
func (p *PathImpl) Dirs() []string {
	return p.dirs
}

// Errors returns an array of all parse and validation errors encountered when parsing.
func (p *PathImpl) Errors() []error {
	return p.errs
}

// Path parses a local or remote file or directory by purely lexical
// processing, and returns an object for use through functional
// semantics.
//
// It is able to parse the following types of input:
//		1. Relative file or directory
//      2. Absolute file or directory
//		3. UNC file or directory
//      4. UNICODE absolute file or directory
//      5. UNICODE UNC file or directory
//
// Errors are collected during the parsing, for all possible
// validation errors describe by the referenced MSDN article later
// described.
//
// See also MSDN, ``Naming Files, Paths, and Namespaces,''
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx
func Path(path string) *PathImpl {
	return newPathImpl(path)
}
