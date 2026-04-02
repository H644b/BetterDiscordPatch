import os
import shutil

os.chdir(os.path.dirname(__file__))
def clear():
    for i in range(2):
        os.system("clear")

def run_sh(sh):
    for cmd in sh.split("\n"):
        os.system(f"{cmd}")

def build(op):
    global branch
    global send_success_notifications

    suffix = ".app" if op == "Darwin" else ".exe"
    branch = "stable"
    send_success_notifications = True

    cli_code = open("./files/cli.go", "r").read()
    cli_code = cli_code.replace("var pyBranch = \"stable\"", f"var pyBranch = \"{branch}\"")
    cli_code = cli_code.replace("var pySendSuccessNotifications = true", f"var pySendSuccessNotifications = {str(send_success_notifications).lower()}")
    open("./installer/cli.go", "w").write(cli_code)

    os.chdir("./installer/")
    build_vi = f"""
    go mod tidy
    CGO_ENABLED=0{" GOOS=windows GOARCH=amd64 " if op == "Windows" else " "}go build{" -ldflags=\"-H=windowsgui\" " if op == "Windows" else " "}--tags cli
    """
    build_vi_darwin = """
    mkdir -p BetterDiscordPatch.app/Contents/MacOS
    mkdir -p BetterDiscordPatch.app/Contents/Resources
    cp macos/Info.plist BetterDiscordPatch.app/Contents/Info.plist
    mv BetterDiscordPatch BetterDiscordPatch.app/Contents/MacOS/BetterDiscordPatch
    cp macos/icon.icns BetterDiscordPatch.app/Contents/Resources/icon.icns
    rm -rf ../BetterDiscordPatch.app
    """
    run_sh(build_vi)
    if op == "Darwin":
        run_sh(build_vi_darwin)
    os.system(f"mv BetterDiscordPatch{suffix} ../BetterDiscordPatch{suffix}")

    os.chdir("../")
    os.remove("./installer/cli.go")
    os.system(f"mv BetterDiscordPatch{suffix} ./binaries/BetterDiscordPatch{suffix}")

clear()
if os.path.exists("./binaries/"):
    shutil.rmtree("./binaries")
os.mkdir("./binaries/")

os.system("cp ./files/autovencordpatch.go ./autopatch/autovencordpatch.go")
os.system("cp ./files/autovencordpatch_win.go ./autopatch/autovencordpatch_win.go")
os.chdir("./autopatch")
build_avp = f"""
go mod tidy
CGO_ENABLED=0 go build -o autodiscordpatch autovencordpatch.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags=\"-H=windowsgui\" -o autodiscordpatch.exe autovencordpatch_win.go
"""
run_sh(build_avp)
os.chdir("../")
os.rename("./autopatch/autodiscordpatch", "./binaries/autodiscordpatch")
os.rename("./autopatch/autodiscordpatch.exe", "./binaries/autodiscordpatch.exe")
os.remove("./autopatch/autovencordpatch.go")
os.remove("./autopatch/autovencordpatch_win.go")

for op in ["Windows", "Darwin"]:
    build(op)
os.system("cp ./autopatch/org.aaron.autodiscordpatch.plist ./binaries/org.aaron.autodiscordpatch.plist")