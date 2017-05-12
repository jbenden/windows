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
package windows_test

import (
	"gitlab.com/jbenden/windows"
	"math/rand"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

// RandStringBytesMaskImprSrc generates a string of a specified length.
//
// "Borrowed" from: http://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
func RandStringBytesMaskImprSrc(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

// ErrString is a helper used with Gomega's WithTransform function, to return the string inside an error.
func ErrString(e error) string {
	return e.Error()
}

func DescribeWhen(text string, when func() bool, body func()) {
	if when() {
		Describe(text, body)
	}
}

var _ = Describe("Path", func() {
	var subject *windows.PathImpl

	Context("when an invalid drive letter is present", func() {
		BeforeEach(func() {
			subject = windows.Path("1:")

			Expect(subject).ShouldNot(BeNil())
		})

		It("should not have the correct disk/device present", func() {
			Expect(subject.Device()).ShouldNot(Equal("1"))
		})

		It("should have errors present", func() {
			Expect(subject.Errors()).ShouldNot(BeEmpty())
		})
	})

	DescribeTable("when an invalid path letter is present",
		func(target string) {
			subject = windows.Path(target)

			Expect(subject).ShouldNot(BeNil())
			Expect(subject.Device()).Should(BeEmpty())
			Expect(subject.Name()).To(Equal(target))
			Expect(subject.Errors()).ShouldNot(BeEmpty())
		},
		Entry("a path with a tab", "a\x09b"),
		Entry("a path with a NULL", "a\x00bc"),
	)

	DescribeTable("when only a drive letter is present",
		func(target string) {
			subject = windows.Path(target)

			Expect(subject).ShouldNot(BeNil())
			Expect(subject.Device()).Should(Equal("C"))
			Expect(subject.Name()).To(BeEmpty())
			Expect(subject.IsAbsolute()).To(BeFalse())
			Expect(subject.IsRelative()).To(BeTrue())
			Expect(subject.ToUnicodeUNC()).To(Equal("\\\\?\\C:\\"))
		},
		Entry("a lower-case drive", "c:"),
		Entry("a upper-case drive", "C:"),
	)

	DescribeTable("when paths that exceed the maximum lengths",
		func(target string, expected bool) {
			subject = windows.Path(target)

			Expect(subject).ShouldNot(BeNil())

			var expecting string
			if strings.HasPrefix(target, "\\\\?\\") {
				expecting = "UNICODE path exceeds"
			} else {
				expecting = "path exceeds"
			}

			if expected {
				Expect(subject.Errors()).Should(ContainElement(WithTransform(ErrString, ContainSubstring(expecting))))
			} else {
				Expect(subject.Errors()).ShouldNot(ContainElement(WithTransform(ErrString, ContainSubstring(expecting))))
			}
		},
		Entry("a non-UNICODE path, just under", RandStringBytesMaskImprSrc(255), false),
		Entry("a non-UNICODE path", RandStringBytesMaskImprSrc(256), true),
		Entry("a UNICODE path, just under", "\\\\?\\"+RandStringBytesMaskImprSrc(32762), false),
		Entry("a UNICODE path", "\\\\?\\"+RandStringBytesMaskImprSrc(34000), true),
	)

	Context("when only a drive letter is present", func() {
		for _, target := range []string{"C:", "c:"} {
			BeforeEach(func() {
				subject = windows.Path(target)

				Expect(subject).ShouldNot(BeNil())
			})

			It("should have the correct disk/device present", func() {
				Expect(subject.Device()).Should(Equal("C"))
			})

			It("should not have a name present", func() {
				Expect(subject.Name()).To(BeEmpty())
			})

			It("should not be an absolute path", func() {
				Expect(subject.IsAbsolute()).To(BeFalse())
				Expect(subject.IsRelative()).To(BeTrue())
			})

			It("should generate an UNICODE UNC path", func() {
				Expect(subject.ToUnicodeUNC()).To(Equal("\\\\?\\C:\\"))
			})
		}
	})

	Context("when a drive letter with trailing backslash is present", func() {
		BeforeEach(func() {
			subject = windows.Path("C:\\")

			Expect(subject).ShouldNot(BeNil())
		})

		It("should have the correct disk/device present", func() {
			Expect(subject.Device()).Should(Equal("C"))
		})

		It("should not have a name present", func() {
			Expect(subject.Name()).To(BeEmpty())
		})

		It("should be an absolute path", func() {
			Expect(subject.IsAbsolute()).To(BeTrue())
			Expect(subject.IsRelative()).NotTo(BeTrue())
		})

		It("should generate an UNICODE UNC path", func() {
			Expect(subject.ToUnicodeUNC()).To(Equal("\\\\?\\C:\\"))
		})
	})

	Context("when a drive letter and path is present", func() {
		BeforeEach(func() {
			subject = windows.Path("C:\\msys64")

			Expect(subject).ShouldNot(BeNil())
		})

		It("should have the correct disk/device present", func() {
			Expect(subject.Device()).Should(Equal("C"))
		})

		It("should have the correct path", func() {
			Expect(subject.Name()).Should(Equal("msys64"))
		})

		Context("when functional programming is utilized", func() {
			It("should force trailing name to a directory entry", func() {
				Expect(subject.MakeDirectory().Dirs()).To(ContainElement("msys64"))
			})

			It("should have an empty file name", func() {
				Expect(subject.MakeDirectory().Name()).To(BeEmpty())
			})

			It("should generate an UNICODE UNC path", func() {
				Expect(subject.MakeDirectory().ToUnicodeUNC()).To(Equal("\\\\?\\C:\\msys64\\"))
			})
		})

		It("should not have any paths", func() {
			Expect(subject.Dirs()).Should(BeEmpty())
		})

		It("should be an absolute path", func() {
			Expect(subject.IsAbsolute()).To(BeTrue())
		})

		It("should generate an UNICODE UNC path", func() {
			Expect(subject.ToUnicodeUNC()).To(Equal("\\\\?\\C:\\msys64"))
		})
	})

	Context("when an absolute path is present", func() {
		BeforeEach(func() {
			subject = windows.Path("\\msys64")

			Expect(subject).ShouldNot(BeNil())
		})

		It("should have the correct path", func() {
			Expect(subject.Name()).Should(Equal("msys64"))
		})

		It("should not have any paths", func() {
			Expect(subject.Dirs()).Should(BeEmpty())
		})

		It("should be an absolute path", func() {
			Expect(subject.IsAbsolute()).To(BeTrue())
		})
	})

	Context("when a UNICODE UNC drive and path is present", func() {
		BeforeEach(func() {
			subject = windows.Path("\\\\?\\C:\\msys64")

			Expect(subject).ShouldNot(BeNil())
		})

		It("should have the correct device/drive", func() {
			Expect(subject.Device()).To(Equal("C"))
		})

		It("should have the correct path", func() {
			Expect(subject.Name()).To(Equal("msys64"))
		})

		It("should be an absolute path", func() {
			Expect(subject.IsAbsolute()).To(BeTrue())
		})

		It("should be a local path", func() {
			Expect(subject.IsLocal()).To(BeTrue())
		})

		It("should not be a remote path", func() {
			Expect(subject.IsRemote()).ToNot(BeTrue())
		})
	})

	Context("when a UNC share and path is present", func() {
		BeforeEach(func() {
			subject = windows.Path("\\\\peaches\\msys64")

			Expect(subject).ShouldNot(BeNil())
		})

		It("should reference the correct node", func() {
			Expect(subject.Node()).To(Equal("peaches"))
		})

		It("should have the correct path name present", func() {
			Expect(subject.Name()).To(Equal("msys64"))
		})

		It("should not be a local path", func() {
			Expect(subject.IsLocal()).ToNot(BeTrue())
		})

		It("should be a remote path", func() {
			Expect(subject.IsRemote()).To(BeTrue())
		})

		It("should be a relative path", func() {
			Expect(subject.IsRelative()).To(BeTrue())
			Expect(subject.IsAbsolute()).ToNot(BeTrue())
		})
	})

	Context("when a UNICODE UNC share and path is present", func() {
		BeforeEach(func() {
			subject = windows.Path("\\\\?\\UNC\\peaches\\msys64")

			Expect(subject).ShouldNot(BeNil())
		})

		It("should reference the correct node", func() {
			Expect(subject.Node()).To(Equal("peaches"))
		})

		It("should have the correct path name present", func() {
			Expect(subject.Name()).To(Equal("msys64"))
		})

		It("should not be a local path", func() {
			Expect(subject.IsLocal()).ToNot(BeTrue())
		})

		It("should be a remote path", func() {
			Expect(subject.IsRemote()).To(BeTrue())
		})

		It("should be a relative path", func() {
			Expect(subject.IsRelative()).To(BeTrue())
			Expect(subject.IsAbsolute()).ToNot(BeTrue())
		})

		It("should be a correct UNICODE UNC path generated", func() {
			Expect(subject.ToUnicodeUNC()).To(Equal("\\\\?\\UNC\\peaches\\msys64"))
		})
	})

	Context("when a UNC share and a longer path is present", func() {
		BeforeEach(func() {
			subject = windows.Path("\\\\peaches\\msys64\\home\\joe")

			Expect(subject).ShouldNot(BeNil())
		})

		It("should reference the correct node", func() {
			Expect(subject.Node()).To(Equal("peaches"))
		})

		It("should have the correct path name present", func() {
			Expect(subject.Name()).To(Equal("joe"))
		})

		It("should have the correct paths present", func() {
			Expect(subject.Dirs()).To(BeEquivalentTo([]string{"msys64", "home"}))
		})

		It("should not be a local path", func() {
			Expect(subject.IsLocal()).ToNot(BeTrue())
		})

		It("should be a remote path", func() {
			Expect(subject.IsRemote()).To(BeTrue())
		})

		It("should be a relative path", func() {
			Expect(subject.IsRelative()).To(BeTrue())
			Expect(subject.IsAbsolute()).ToNot(BeTrue())
		})

		It("should be a correct UNICODE UNC path generated", func() {
			Expect(subject.ToUnicodeUNC()).To(Equal("\\\\?\\UNC\\peaches\\msys64\\home\\joe"))
		})

		It("should be a correct UNC path generated", func() {
			Expect(subject.ToString()).To(Equal("\\\\peaches\\msys64\\home\\joe"))
		})
	})

	DescribeWhen("running on maintainers machine",
		func() bool {
			if name, ok := windows.ComputerName(); ok == nil {
				return name == "WINDOWS-F84BCIB"
			}
			return false
		},
		func() {
			Context("the machine name", func() {
				It("should be the expected value", func() {
					name, err := windows.ComputerName()
					Expect(err).ShouldNot(HaveOccurred())
					Expect(name).To(BeIdenticalTo("WINDOWS-F84BCIB"))
				})
			})

			Context("the system directory", func() {
				It("should be the expected value", func() {
					name, err := windows.SystemDirectory()
					Expect(err).ShouldNot(HaveOccurred())
					Expect(name).To(BeIdenticalTo("C:\\WINDOWS\\system32"))
				})
			})

			Context("the system configuration directory", func() {
				It("should be the expected value", func() {
					name, err := windows.ConfigDirectory()
					Expect(err).ShouldNot(HaveOccurred())
					Expect(name).To(BeIdenticalTo("C:\\ProgramData"))
				})
			})

			Context("the user home directory", func() {
				It("should be the expected value", func() {
					name, err := windows.HomeDirectory()
					Expect(err).ShouldNot(HaveOccurred())
					Expect(name).To(BeIdenticalTo("C:\\Users\\Joseph Benden"))
				})
			})

			Context("the user configuration directory", func() {
				It("should be the expected value", func() {
					name, err := windows.ConfigHomeDirectory()
					Expect(err).ShouldNot(HaveOccurred())
					Expect(name).To(BeIdenticalTo("C:\\Users\\Joseph Benden\\AppData\\Roaming"))
				})
			})

			Context("the user data configuration directory", func() {
				It("should be the expected value", func() {
					name, err := windows.DataHomeDirectory()
					Expect(err).ShouldNot(HaveOccurred())
					Expect(name).To(BeIdenticalTo("C:\\Users\\Joseph Benden\\AppData\\Local"))
				})
			})

			Context("when an absolute path is forced to absolute", func() {
				BeforeEach(func() {
					subject = windows.Path("\\msys64").MakeAbsolute()
				})

				It("should not have a disk/device present", func() {
					Expect(subject.Device()).Should(BeEmpty())
				})

				It("should have the correct path", func() {
					Expect(subject.Name()).Should(Equal("msys64"))
				})

				It("should be an absolute path", func() {
					Expect(subject.IsAbsolute()).To(BeTrue())
				})

				It("should exist in the file system", func() {
					Expect(subject.ToString()).To(Equal("\\msys64"))
					Expect(subject.IsDirectoryExists()).To(BeTrue())
				})
			})

			Context("when a file is forced to be an absolute path", func() {
				BeforeEach(func() {
					subject = windows.Path("path_test.go").MakeAbsolute()
				})

				It("should have the correct disk/device present", func() {
					Expect(subject.Device()).Should(Equal("C"))
				})

				It("should have the correct file name", func() {
					Expect(subject.Name()).Should(Equal("path_test.go"))
				})

				It("should be an absolute path", func() {
					Expect(subject.IsAbsolute()).To(BeTrue())
				})

				It("should fail a directory exists test", func() {
					Expect(subject.IsDirectoryExists()).NotTo(BeTrue())
				})
			})

		},
	)
})
