/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"os"
	path "path/filepath"
	"strings"
)

// macosDataNames maps branch name to the Discord user-data directory name under
// ~/Library/Application Support/. These are where Discord stores its modules.
var macosDataNames = map[string]string{
	"stable": "discord",
	"ptb":    "discordptb",
	"canary": "discordcanary",
	"dev":    "discorddevelopment",
}

// macosAppNames maps branch name to the application bundle name in /Applications/.
// This is used to locate the resources directory for OpenAsar support.
var macosAppNames = map[string]string{
	"stable": "Discord.app",
	"ptb":    "Discord PTB.app",
	"canary": "Discord Canary.app",
	"dev":    "Discord Development.app",
}

// ParseDiscord finds the discord_desktop_core module directory within a Discord
// user-data directory (e.g. ~/Library/Application Support/discord/).
// Version subdirectories have the format "0.0.298" (multiple dot-separated numbers).
// As a convenience, if p is "/Applications" or "/Applications/", the function
// scans /Applications/ for known Discord app bundles and tries the corresponding
// ~/Library/Application Support/ data directory for each one found.
func ParseDiscord(p, branch string) *DiscordInstall {
	if path.Clean(p) == "/Applications" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		appSupport := path.Join(home, "Library", "Application Support")
		for b, dataName := range macosDataNames {
			if branch != "" && branch != b && branch != "auto" {
				continue
			}
			appBundle, ok := macosAppNames[b]
			if !ok || !ExistsFile(path.Join("/Applications", appBundle)) {
				continue
			}
			dataPath := path.Join(appSupport, dataName)
			if di := ParseDiscord(dataPath, b); di != nil {
				return di
			}
		}
		return nil
	}

	if !ExistsFile(p) {
		return nil
	}

	entries, err := os.ReadDir(p)
	if err != nil {
		return nil
	}

	appPath := ""
	resourcesPath := ""
	isPatched := false

	for _, versionDir := range entries {
		if !versionDir.IsDir() {
			continue
		}
		// Version directories look like "0.0.298" - must have at least two dots
		if strings.Count(versionDir.Name(), ".") < 2 {
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
		for _, mdir := range moduleEntries {
			if !mdir.IsDir() || !strings.HasPrefix(mdir.Name(), "discord_desktop_core") {
				continue
			}
			coreDir := path.Join(modulesPath, mdir.Name(), "discord_desktop_core")
			if !ExistsFile(path.Join(coreDir, "core.asar")) {
				continue
			}
			if bestCoreDir == "" || mdir.Name() > path.Base(path.Dir(bestCoreDir)) {
				bestCoreDir = coreDir
			}
		}

		// Keep the candidate from the latest version directory (proper numeric comparison)
		if bestCoreDir != "" && (appPath == "" || versionGreater(versionDir.Name(), path.Base(path.Dir(path.Dir(path.Dir(appPath)))))) {
			appPath = bestCoreDir
			isPatched = isInjected(bestCoreDir)
		}
	}

	if appPath == "" {
		return nil
	}

	if branch == "" {
		branch = GetBranch(strings.ToLower(path.Base(p)))
	}

	// Resources are inside the application bundle in /Applications/
	if appBundle, ok := macosAppNames[branch]; ok {
		resourcesPath = path.Join("/Applications", appBundle, "Contents", "Resources")
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

	home, err := os.UserHomeDir()
	if err != nil {
		Log.Error("Failed to get home directory:", err)
		return discords
	}
	appSupport := path.Join(home, "Library", "Application Support")

	for branch, dirname := range macosDataNames {
		p := path.Join(appSupport, dirname)
		if discord := ParseDiscord(p, branch); discord != nil {
			Log.Debug("Found Discord Install at", p)
			discords = append(discords, discord)
		}
	}
	return discords
}

func PreparePatch(di *DiscordInstall) {}

func FixOwnership(_ string) error {
	return nil
}

func CheckScuffedInstall() bool {
	return false
}
