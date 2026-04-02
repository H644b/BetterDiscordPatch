import os
import platform

os.chdir(os.path.dirname(__file__))
def clear():
    for i in range(2):
        os.system("clear")

def run_sh(sh):
    for cmd in sh.split("\n"):
        os.system(f"{cmd}")

clear()
print("[BetterDiscordPatch Installer (macOS)]")
branch = input("Enter the branch of Discord to be patched by BetterDiscord (stable, ptb, canary): ")
if branch not in ["stable", "ptb", "canary"]:
    input("This branch of Discord doesn't exist. ")
    exit()
use_autopatch = input("Patch this branch of Discord through updates (y/n)? ").lower().strip() == "y"
send_success_notifications = input("Send notifications on success (y/n)? ").lower().strip() == "y"

clear()
print("[Installing BetterDiscordPatch]")
print(f"Installing with preferences: branch='{branch}', use_autopatch={use_autopatch}, send_success_notifications={send_success_notifications}")
print("\nRunning pre-install checks...", end=" ", flush=True)
if platform.system() != "Darwin":
    print("failed")
    input("This operating system is not supported by this installer. ")
    exit()
for dir in ["./files/", "./autopatch/" if use_autopatch else "./files/", "./installer/"]:
    if not os.path.exists(dir):
        print("failed")
        input(f"The directory '{dir}' is missing. ")
        exit()
for file in ["./files/autovencordpatch.go" if use_autopatch else "./files/cli.go", "./files/cli.go"]:
    if not os.path.exists(file):
        print("failed")
        input(f"The file '{file}' is missing. ")
        exit()
print("done")

discords = {
    "stable": "Discord.app",
    "ptb": "Discord PTB.app",
    "canary": "Discord Canary.app"
}
print("Preparing install environment...", end=" ", flush=True)
if use_autopatch:
    avp_code = open("./files/autovencordpatch.go", "r").read()
    avp_code = avp_code.replace("Discord.app", discords[branch])
    open("./autopatch/autovencordpatch.go", "w").write(avp_code)
cli_code = open("./files/cli.go", "r").read()
cli_code = cli_code.replace("var pyBranch = \"stable\"", f"var pyBranch = \"{branch}\"")
cli_code = cli_code.replace("var pySendSuccessNotifications = true", f"var pySendSuccessNotifications = {str(send_success_notifications).lower()}")
open("./installer/cli.go", "w").write(cli_code)
print("done")

os.chdir("./installer/")
build_vi = """
go mod tidy
CGO_ENABLED=0 go build --tags cli
mkdir -p BetterDiscordPatch.app/Contents/MacOS
mkdir -p BetterDiscordPatch.app/Contents/Resources
cp macos/Info.plist BetterDiscordPatch.app/Contents/Info.plist
mv BetterDiscordPatch BetterDiscordPatch.app/Contents/MacOS/BetterDiscordPatch
cp macos/icon.icns BetterDiscordPatch.app/Contents/Resources/icon.icns
rm -rf ../BetterDiscordPatch.app
mv BetterDiscordPatch.app ../BetterDiscordPatch.app
"""
print("Building BetterDiscordPatch.app...", end=" ", flush=True)
run_sh(build_vi)
print("done")

if use_autopatch:
    print("Building auto-patch binary...", end=" ", flush=True)
    os.chdir("../autopatch/")
    build_avp = """
    go mod tidy
    CGO_ENABLED=0 go build -o autodiscordpatch autovencordpatch.go
    chmod +x autodiscordpatch
    mv autodiscordpatch ../BetterDiscordPatch.app/Contents/Resources/autodiscordpatch
    """
    run_sh(build_avp)
    print("done")

os.chdir("../")
mv_to_applications = """
rm -rf /Applications/BetterDiscordPatch.app
mv BetterDiscordPatch.app /Applications/BetterDiscordPatch.app
"""
run_sh(mv_to_applications)

if use_autopatch:
    print("Running auto-patch install scripts...", end=" ", flush=True)
    install = """
    cp autopatch/org.aaron.autovencordpatch.plist ~/Library/LaunchAgents/org.aaron.autodiscordpatch.plist
    launchctl unload ~/Library/LaunchAgents/org.aaron.autodiscordpatch.plist > /dev/null 2>&1
    launchctl load ~/Library/LaunchAgents/org.aaron.autodiscordpatch.plist > /dev/null 2>&1
    open /Applications/BetterDiscordPatch.app
    """
    run_sh(install)
    print("done")

print("Cleaning up...", end=" ", flush=True)
os.remove("./installer/cli.go")
if use_autopatch:
    os.remove("./autopatch/autovencordpatch.go")
print("done")

input("\nSuccessfully installed BetterDiscordPatch! ")