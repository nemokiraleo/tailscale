// Copyright (c) 2020 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hostinfo

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"tailscale.com/syncs"
	"tailscale.com/util/winutil"
)

func init() {
	osVersion = osVersionWindows
	packageType = packageTypeWindows
}

var winVerCache syncs.AtomicValue[string]

func osVersionWindows() string {
	if s, ok := winVerCache.LoadOk(); ok {
		return s
	}
	major, minor, build := windows.RtlGetNtVersionNumbers()
	s := fmt.Sprintf("%d.%d.%d", major, minor, build)
	// Windows 11 still uses 10 as its major number internally
	if major == 10 {
		if ubr, err := getUBR(); err == nil {
			s += fmt.Sprintf(".%d", ubr)
		}
	}
	if s != "" {
		winVerCache.Store(s)
	}
	return s // "10.0.19041.388", ideally
}

// getUBR obtains a fourth version field, the "Update Build Revision",
// from the registry. This field is only available beginning with Windows 10.
func getUBR() (uint32, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return 0, err
	}
	defer key.Close()

	val, valType, err := key.GetIntegerValue("UBR")
	if err != nil {
		return 0, err
	}
	if valType != registry.DWORD {
		return 0, registry.ErrUnexpectedType
	}

	return uint32(val), nil
}

func packageTypeWindows() string {
	if _, err := os.Stat(`C:\ProgramData\chocolatey\lib\tailscale`); err == nil {
		return "choco"
	}
	if msiSentinel := winutil.GetRegInteger("MSI", 0); msiSentinel == 1 {
		return "msi"
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exe)
	nsisUninstaller := filepath.Join(dir, "Uninstall-Tailscale.exe")
	_, err = os.Stat(nsisUninstaller)
	if err == nil {
		return "nsis"
	}
	// Atypical. Not worth trying to detect. Likely open
	// source tailscaled or a developer running by hand.
	return ""
}
