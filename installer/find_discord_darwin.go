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
// If p is a .app bundle path (e.g. /Applications/Discord.app), the function
// maps it to the corresponding data directory automatically.
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

	// If the user passed a .app bundle path, map it to the data directory.
	cleanP := path.Clean(p)
	if strings.HasSuffix(strings.ToLower(cleanP), ".app") {
		appBase := path.Base(cleanP)
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		appSupport := path.Join(home, "Library", "Application Support")
		for b, appName := range macosAppNames {
			if !strings.EqualFold(appName, appBase) {
				continue
			}
			if branch != "" && branch != b && branch != "auto" {
				continue
			}
			dataPath := path.Join(appSupport, macosDataNames[b])
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
	bestVersion := ""
	resourcesPath := ""
	isPatched := false

	for _, versionDir := range entries {
		if !versionDir.IsDir() {
			continue
		}
		// Version directories look like "0.0.298" or "0.308" - require at least one dot
		// so that plain names like "modules" or "storage" are skipped.
		if strings.Count(versionDir.Name(), ".") < 1 {
			Log.Debug("Skipping non-version directory:", versionDir.Name())
			continue
		}

		versionPath := path.Join(p, versionDir.Name())
		modulesPath := path.Join(versionPath, "modules")
		if !ExistsFile(modulesPath) {
			Log.Debug("No modules dir in version directory:", versionPath)
			continue
		}

		moduleEntries, err := os.ReadDir(modulesPath)
		if err != nil {
			Log.Debug("Could not read modules dir:", modulesPath, err)
			continue
		}

		bestCoreDir := ""
		bestModuleDir := ""
		for _, mdir := range moduleEntries {
			if !mdir.IsDir() || !strings.HasPrefix(mdir.Name(), "discord_desktop_core") {
				continue
			}
			// Try the standard nested structure first:
			// discord_desktop_core-N/discord_desktop_core/core.asar
			nested := path.Join(modulesPath, mdir.Name(), "discord_desktop_core")
			if ExistsFile(path.Join(nested, "core.asar")) {
				if bestCoreDir == "" || mdir.Name() > bestModuleDir {
					bestCoreDir = nested
					bestModuleDir = mdir.Name()
				}
				continue
			}
			// Fallback: some installs place core.asar directly in the module dir
			// discord_desktop_core-N/core.asar
			flat := path.Join(modulesPath, mdir.Name())
			if ExistsFile(path.Join(flat, "core.asar")) {
				Log.Debug("Found core.asar in flat module dir:", flat)
				if bestCoreDir == "" || mdir.Name() > bestModuleDir {
					bestCoreDir = flat
					bestModuleDir = mdir.Name()
				}
				continue
			}
			Log.Debug("discord_desktop_core dir found but core.asar missing:", path.Join(modulesPath, mdir.Name()))
		}

		// Keep the candidate from the latest version directory (proper numeric comparison).
		// Track bestVersion explicitly to avoid deriving it from path depth (which differs
		// between the nested and flat module structures).
		if bestCoreDir != "" && (appPath == "" || versionGreater(versionDir.Name(), bestVersion)) {
			appPath = bestCoreDir
			bestVersion = versionDir.Name()
			isPatched = isInjected(bestCoreDir)
		}
	}

	if appPath == "" {
		Log.Debug("No discord_desktop_core with core.asar found under:", p)
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
