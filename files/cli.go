//go:build cli

/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/fatih/color"
)

var discords []any

func isValidBranch(branch string) bool {
	switch branch {
	case "", "stable", "ptb", "canary", "auto":
		return true
	default:
		return false
	}
}

func die(msg string) {
	Log.Error(msg)
	exitFailure(msg)
}

var pyBranch = "stable"
var pySendSuccessNotifications = true

func main() {
	InitGithubDownloader()
	discords = FindDiscords()

	// Used by log.go init func
	flag.Bool("debug", false, "Enable debug info")

	var versionFlag = flag.Bool("version", false, "View the program version")
	var installFlag = flag.Bool("install", true, "Install BetterDiscord")
	var updateFlag = flag.Bool("repair", false, "Repair BetterDiscord")
	var uninstallFlag = flag.Bool("uninstall", false, "Uninstall BetterDiscord")
	var dirFlag = flag.String("dir", "", "The location of the Discord install to modify")
	var branchFlag = flag.String("branch", pyBranch, "The branch of Discord to modify [auto|stable|ptb|canary]")
	flag.Parse()

	if *versionFlag {
		fmt.Println("BetterDiscordPatch v0.3.0")
		fmt.Println("Modified to install BetterDiscord without user interaction")
		return
	}

	if !isValidBranch(*branchFlag) {
		die("The 'branch' flag must be one of the following: [auto|stable|ptb|canary]")
	}

	if *installFlag || *updateFlag {
		if !<-GithubDoneChan {
			die("Not " + Ternary(*installFlag, "installing", "updating") + " as fetching release data failed.")
		}
	}

	install, uninstall, update := *installFlag, *uninstallFlag, *updateFlag

	var err error
	var errSilent error
	if install {
		errSilent = PromptDiscord("patch", *dirFlag, *branchFlag).patch()
	} else if uninstall {
		errSilent = PromptDiscord("unpatch", *dirFlag, *branchFlag).unpatch()
	} else if update {
		Log.Info("Downloading latest BetterDiscord files...")
		err := installLatestBuilds()
		Log.Info("Done!")
		if err == nil {
			errSilent = PromptDiscord("repair", *dirFlag, *branchFlag).patch()
		}
	}

	if err != nil {
		Log.Error(err)
		exitFailure(err.Error())
	}
	if errSilent != nil {
		exitFailure()
	}

	exitSuccess()
}

func exitSuccess() {
	if pySendSuccessNotifications == true {
		if runtime.GOOS == "darwin" {
			notify("BetterDiscordPatch", "Successfully installed BetterDiscord!")
		} else {
			notify("Success", "Successfully installed BetterDiscord!")
		}
	}
	color.HiGreen("Success!")
	os.Exit(0)
}

func exitFailure(reason ...string) {
	displayed_reason := "Failed to patch BetterDiscord"
	if len(reason) > 0 {
		displayed_reason = "Failed to patch BetterDiscord: " + reason[0]
	}
	color.HiRed("Failed!")

	if runtime.GOOS == "darwin" {
		notify("BetterDiscordPatch", displayed_reason)
	} else {
		notify("An error has occurred.", displayed_reason)
	}
	os.Exit(1)
}

func PromptDiscord(action, dir, branch string) *DiscordInstall {
	if dir != "" {
		install := ParseDiscord(dir, branch)
		if install == nil {
			die("No Discord install was found at the specified directory.")
		}
		return install
	}

	for _, discord := range discords {
		install := discord.(*DiscordInstall)
		if branch == "auto" || install.branch == branch {
			return install
		}
	}

	// Fallback: try /Applications/ on macOS before giving up
	if runtime.GOOS == "darwin" {
		if install := ParseDiscord("/Applications/", branch); install != nil {
			return install
		}
	}

	die("No Discord install was found. Try manually specifying the directory with --dir.")
	return nil
}

func InstallLatestBuilds() error {
	return installLatestBuilds()
}

func HandleScuffedInstall() {
	fmt.Println("Hold on!")
	fmt.Println("You have a broken Discord install.")
	fmt.Println("Please reinstall Discord before proceeding!")
	fmt.Println("Otherwise, BetterDiscord will likely not work.")
}
