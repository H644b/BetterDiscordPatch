/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"encoding/json"
	"errors"
	"os"
	path "path/filepath"
	"strings"

	"github.com/ProtonMail/go-appdir"
)

var BaseDir string
var FilesDir string
var FilesDirErr error
var Patcher string

func init() {
	if dir := os.Getenv("BD_USER_DATA_DIR"); dir != "" {
		Log.Debug("Using BD_USER_DATA_DIR")
		BaseDir = dir
	} else {
		Log.Debug("Using UserConfig")
		BaseDir = appdir.New("BetterDiscord").UserConfig()
	}
	FilesDir = path.Join(BaseDir, "data")
	if !ExistsFile(FilesDir) {
		FilesDirErr = os.MkdirAll(FilesDir, 0755)
		if FilesDirErr != nil {
			Log.Error("Failed to create", FilesDir, FilesDirErr)
		} else {
			FilesDirErr = FixOwnership(BaseDir)
		}
	}
	Patcher = path.Join(FilesDir, "betterdiscord.asar")
}

type DiscordInstall struct {
	path             string // the base path
	branch           string // canary / stable / ...
	appPath          string // discord_desktop_core directory to inject index.js into
	resourcesPath    string // resources directory (used for OpenAsar)
	isPatched        bool
	isFlatpak        bool
	isSystemElectron bool // Needs special care https://aur.archlinux.org/packages/discord_arch_electron
	isOpenAsar       *bool
}

// isInjected reports whether the discord_desktop_core directory already has a BD shim.
func isInjected(dir string) bool {
	indexJs := path.Join(dir, "index.js")
	content, err := os.ReadFile(indexJs)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), "betterdiscord.asar")
}

//region Patch

// injectShim writes the BetterDiscord loader shim into discord_desktop_core/index.js.
// The shim requires betterdiscord.asar and then re-exports the original core.asar,
// which is how the official BetterDiscord installer injects BD into Discord.
func injectShim(dir string) error {
	indexJs := path.Join(dir, "index.js")
	patcherPathB, _ := json.Marshal(Patcher)
	content := "require(" + string(patcherPathB) + ");\nmodule.exports = require(\"./core.asar\");"
	Log.Debug("Writing shim to", indexJs)
	if err := os.WriteFile(indexJs, []byte(content), 0644); err != nil {
		err = CheckIfErrIsCauseItsBusyRn(err)
		return err
	}
	return nil
}

func (di *DiscordInstall) patch() error {
	Log.Info("Patching " + di.path + "...")
	if LatestHash != InstalledHash {
		if err := InstallLatestBuilds(); err != nil {
			return err // already shown dialog so don't return same error again
		}
	}

	PreparePatch(di)

	if di.isPatched {
		Log.Info(di.path, "is already patched. Updating shim...")
	}

	if err := injectShim(di.appPath); err != nil {
		if errors.Is(err, os.ErrPermission) {
			notify("BetterDiscordPatch", "The App Management/Full Disk Access permission must be granted to allow BetterDiscordPatch to patch BetterDiscord. Make sure Discord isn't running!")
			os.Exit(1)
			return err
		}
		return err
	}

	Log.Info("Successfully patched", di.path)
	di.isPatched = true

	return nil
}

//endregion

// region Unpatch

// removeShim restores discord_desktop_core/index.js to its original unmodified state.
func removeShim(dir string) (errOut error) {
	indexJs := path.Join(dir, "index.js")
	Log.Debug("Restoring", indexJs)
	if err := os.WriteFile(indexJs, []byte("module.exports = require(\"./core.asar\");"), 0644); err != nil {
		err = CheckIfErrIsCauseItsBusyRn(err)
		Log.Error(err.Error())
		return err
	}
	return nil
}

func (di *DiscordInstall) unpatch() error {
	Log.Info("Unpatching " + di.path + "...")

	PreparePatch(di)

	if err := removeShim(di.appPath); err != nil {
		return err
	}

	Log.Info("Successfully unpatched", di.path)
	di.isPatched = false
	return nil
}

//endregion
