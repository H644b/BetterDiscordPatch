import os
import json
import shutil
import zipfile
import getpass
import platform
import requests
from sys import exit

os.chdir(os.path.dirname(__file__))
def clear():
    os.system("cls" if platform.system() == "Windows" else "clear;clear")

clear()
print("[BetterDiscordPatch Installer]")
print("This installer will download the latest files from GitHub.")
print("")
autopatch = input("Automatically patch Discord with BetterDiscord through updates (y/n)? ").lower().strip() == "y"

releases = requests.get("https://api.github.com/repos/H644b/BetterDiscordPatch/releases")
if not releases.ok:
    print("\nCouldn't fetch releases. Exiting...")
    exit()

paths = {
    "Windows": [
        f"C:/Users/{getpass.getuser()}/AppData/Local/BetterDiscordPatch/betterdiscordpatch.exe",
        f"C:/Users/{getpass.getuser()}/AppData/Roaming/Microsoft/Windows/Start Menu/Programs/Startup/autodiscordpatch.exe",
    ],
    "Darwin": [
        f"/Applications/BetterDiscordPatch.app",
        f"/Applications/BetterDiscordPatch.app/Contents/Resources/autodiscordpatch",
    ],
}

if platform.system() == "Windows":
    os.system("taskkill /f /im autodiscordpatch.exe >NUL 2>&1")
    os.makedirs(f"C:/Users/{getpass.getuser()}/AppData/Local/BetterDiscordPatch/", exist_ok=True)

clear()
print("[Downloading and moving required files...]")
rel = json.loads(releases.text)
for asset in rel[0]["assets"]:
    if platform.system() == "Darwin":
        if f"BetterDiscordPatch.app.zip" == asset["name"]:
            open("BetterDiscordPatch.app.zip", "wb").write(requests.get(asset["browser_download_url"]).content)
            if os.path.exists("/Applications/BetterDiscordPatch.app"):
                shutil.rmtree("/Applications/BetterDiscordPatch.app")
            with zipfile.ZipFile("BetterDiscordPatch.app.zip", 'r') as zip_ref:
                zip_ref.extractall("/Applications/")
            os.system("chmod +x /Applications/BetterDiscordPatch.app/Contents/MacOS/BetterDiscordPatch")
            os.remove("BetterDiscordPatch.app.zip")
            print(f"Successfully downloaded BetterDiscordPatch")
    elif platform.system() == "Windows":
        if f"betterdiscordpatch.exe" == asset["name"]:
            open(f"C:/Users/{getpass.getuser()}/AppData/Local/BetterDiscordPatch/betterdiscordpatch.exe", "wb").write(requests.get(asset["browser_download_url"]).content)
            print(f"Successfully downloaded BetterDiscordPatch")
        elif f"autodiscordpatch.exe" == asset["name"] and autopatch:
            open(f"C:/Users/{getpass.getuser()}/AppData/Roaming/Microsoft/Windows/Start Menu/Programs/Startup/autodiscordpatch.exe", "wb").write(requests.get(asset["browser_download_url"]).content)
            print(f"Successfully installed autopatch component")

if platform.system() == "Darwin":
    for asset in rel[0]["assets"]:
        if asset["name"] == "org.aaron.autodiscordpatch.plist":
            open(f"/Users/{getpass.getuser()}/Library/LaunchAgents/org.aaron.autodiscordpatch.plist", "wb").write(requests.get(asset["browser_download_url"]).content)
            print(f"Successfully installed autopatch launchd plist (macOS)")
        elif asset["name"] == "autodiscordpatch" and autopatch:
            open(f"/Applications/BetterDiscordPatch.app/Contents/Resources/autodiscordpatch", "wb").write(requests.get(asset["browser_download_url"]).content)
            os.system("chmod +x /Applications/BetterDiscordPatch.app/Contents/Resources/autodiscordpatch")
            print(f"Successfully installed autopatch component")
    os.system("open /Applications/BetterDiscordPatch.app")

print("\nSuccessfully installed BetterDiscordPatch!")
input("If you're on Windows and installed the auto-patcher, make sure to restart your computer so the auto-patcher can run. ")
exit()