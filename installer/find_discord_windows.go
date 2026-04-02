/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"errors"
	"golang.org/x/sys/windows"
	"os"
	path "path/filepath"
	"strings"
	"sync"
	"unsafe"
)

var windowsNames = map[string]string{
	"stable": "Discord",
	"ptb":    "DiscordPTB",
	"canary": "DiscordCanary",
	"dev":    "DiscordDevelopment",
}

var killLock sync.Mutex

// ParseDiscord finds the discord_desktop_core module directory within a Discord
// installation base path, which is where BetterDiscord's shim must be injected.
// This mirrors the official BetterDiscord installer's approach.
func ParseDiscord(p, branch string) *DiscordInstall {
	entries, err := os.ReadDir(p)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			Log.Warn("Error during readdir "+p+":", err)
		}
		return nil
	}

	appPath := ""
	resourcesPath := ""
	isPatched := false

	for _, versionDir := range entries {
		if !versionDir.IsDir() || !strings.HasPrefix(versionDir.Name(), "app-") {
			continue
		}

		versionPath := path.Join(p, versionDir.Name())
		modulesPath := path.Join(versionPath, "modules")
		if !ExistsFile(modulesPath) {
			continue
		}

		moduleEntries, err := os.ReadDir(modulesPath)
		if err != nil {
			continue
		}

		bestCoreDir := ""
		bestResources := ""
		for _, mdir := range moduleEntries {
			if !mdir.IsDir() || !strings.HasPrefix(mdir.Name(), "discord_desktop_core") {
				continue
			}
			coreDir := path.Join(modulesPath, mdir.Name(), "discord_desktop_core")
			if !ExistsFile(path.Join(coreDir, "core.asar")) {
				continue
			}
			// Pick the highest-numbered discord_desktop_core-N variant
			if bestCoreDir == "" || mdir.Name() > path.Base(path.Dir(bestCoreDir)) {
				bestCoreDir = coreDir
				bestResources = path.Join(versionPath, "resources")
			}
		}

		// Keep the candidate from the latest app-x.y.z version directory
		if bestCoreDir != "" && (appPath == "" || versionGreater(versionDir.Name(), path.Base(path.Dir(path.Dir(path.Dir(appPath)))))) {
			appPath = bestCoreDir
			resourcesPath = bestResources
			isPatched = isInjected(bestCoreDir)
		}
	}

	if appPath == "" {
		return nil
	}

	if branch == "" {
		branch = GetBranch(p)
	}

	return &DiscordInstall{
		path:             p,
		branch:           branch,
		appPath:          appPath,
		resourcesPath:    resourcesPath,
		isPatched:        isPatched,
		isFlatpak:        false,
		isSystemElectron: false,
	}
}

func FindDiscords() []any {
	var discords []any

	appData := os.Getenv("LOCALAPPDATA")
	if appData == "" {
		Log.Error("%LOCALAPPDATA% is empty???????")
		return discords
	}

	for branch, dirname := range windowsNames {
		p := path.Join(appData, dirname)
		if discord := ParseDiscord(p, branch); discord != nil {
			Log.Debug("Found Discord install at ", p)
			discords = append(discords, discord)
		}
	}
	return discords
}

func PreparePatch(di *DiscordInstall) {
	killLock.Lock()
	defer killLock.Unlock()
	
	name := windowsNames[di.branch]
	Log.Debug("Trying to kill", name)
	pid := findProcessIdByName(name + ".exe")
	if pid == 0 {
		Log.Debug("Didn't find process matching name")
		return
	}

	proc, err := os.FindProcess(int(pid))
	if err != nil {
		Log.Warn("Failed to find process with pid", pid)
		return
	}

	err = proc.Kill()
	if err != nil {
		Log.Warn("Failed to kill", name+":", err)
	} else {
		Log.Debug("Waiting for", name, "to exit")
		_, _ = proc.Wait()
	}
}

func FixOwnership(_ string) error {
	return nil
}

// https://github.com/Vencord/Installer/issues/9

func CheckScuffedInstall() bool {
	username := os.Getenv("USERNAME")
	programData := os.Getenv("PROGRAMDATA")
	for _, discordName := range windowsNames {
		if ExistsFile(path.Join(programData, username, discordName)) || ExistsFile(path.Join(programData, username, discordName)) {
			HandleScuffedInstall()
			return true
		}
	}
	return false
}

func findProcessIdByName(name string) uint32 {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0
	}

	procEntry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for {
		err = windows.Process32Next(snapshot, &procEntry)
		if err != nil {
			return 0
		}
		if windows.UTF16ToString(procEntry.ExeFile[:]) == name {
			return procEntry.ProcessID
		}
	}
}
