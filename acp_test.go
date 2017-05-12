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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ACP", func() {
	Context("when converting with the active code page", func() {
		BeforeSuite(func() {
			// CRITICAL: This test REQUIRE the local machine to be Windows-1252/CP-1252.
			if cp := windows.GetSystemCodePage(); cp != windows.CP1252 {
				Fail("These tests must be ran with CP-1252 as the default Windows Code Page")
			}
		})

		It("should convert CP-1252 to UTF-8", func() {
			actual, err := windows.SystemCodePageToUtf8("hello \x80 world")

			Expect(err).ShouldNot(HaveOccurred())
			Expect(actual).To(BeIdenticalTo("hello \xe2\x82\xac world"))

			/*
				Unable to test for error conditions with CP-1252 to UTF-8. For additional
				information, see: https://en.wikipedia.org/wiki/Windows-1252#Code_page_layout
			*/
		})

		It("should convert UTF-8 to CP-1252", func() {
			actual, err := windows.Utf8ToSystemCodePage("hello \xe2\x82\xac world")

			Expect(err).ShouldNot(HaveOccurred())
			Expect(actual).To(BeIdenticalTo("hello \x80 world"))
		})

		It("should error when converting invalid UTF-8 to CP-1252", func() {
			_, err := windows.Utf8ToSystemCodePage("hello \x7f\xff\xff world")

			Expect(err).Should(HaveOccurred())
		})
	})
})
