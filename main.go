package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	getinfoCmd := flag.NewFlagSet("getinfo", flag.ExitOnError)
	installCmd := flag.NewFlagSet("install", flag.ExitOnError)
	uninstallCmd := flag.NewFlagSet("uninstall", flag.ExitOnError)
	screenshotCmd := flag.NewFlagSet("screenshot", flag.ExitOnError)
	rebootCmd := flag.NewFlagSet("reboot", flag.ExitOnError)
	launchCmd := flag.NewFlagSet("launch", flag.ExitOnError)
	terminateCmd := flag.NewFlagSet("terminate", flag.ExitOnError)
	listProcCmd := flag.NewFlagSet("list-processes", flag.ExitOnError)
	killCmd := flag.NewFlagSet("kill", flag.ExitOnError)
	batteryCmd := flag.NewFlagSet("battery", flag.ExitOnError)
	lsCmd := flag.NewFlagSet("ls", flag.ExitOnError)
	pingCmd := flag.NewFlagSet("ping", flag.ExitOnError)
	
	copyCmd := flag.NewFlagSet("copy", flag.ExitOnError)
	pcFlag := copyCmd.String("pc", "", "Source file on PC to push to mobile")
	mobileFlag := copyCmd.String("mobile", "", "Source file on mobile to pull to PC")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	client := NewClient()
	client.VerifyDeviceConnectionAndVersion(cmd)

	switch cmd {
	case "getinfo":
		getinfoCmd.Parse(os.Args[2:])
		client.HandleGetInfo()
	case "install":
		installCmd.Parse(os.Args[2:])
		args := installCmd.Args()
		if len(args) < 1 {
			fmt.Println("Error: Missing <file.appx> argument.")
			os.Exit(1)
		}
		client.HandleInstall(args[0])
	case "list-apps":
		client.HandleListApps()
	case "uninstall":
		uninstallCmd.Parse(os.Args[2:])
		args := uninstallCmd.Args()
		if len(args) < 1 {
			fmt.Println("Error: Missing <PackageFullName> argument.")
			os.Exit(1)
		}
		client.HandleUninstall(args[0])
	case "screenshot":
		screenshotCmd.Parse(os.Args[2:])
		args := screenshotCmd.Args()
		if len(args) < 1 {
			fmt.Println("Error: Missing <output.png> argument.")
			os.Exit(1)
		}
		client.HandleScreenshot(args[0])
	case "reboot":
		rebootCmd.Parse(os.Args[2:])
		client.HandleReboot()
	case "launch":
		launchCmd.Parse(os.Args[2:])
		args := launchCmd.Args()
		if len(args) < 1 {
			fmt.Println("Error: Missing <AUMID> argument.")
			os.Exit(1)
		}
		client.HandleLaunch(args[0])
	case "terminate":
		terminateCmd.Parse(os.Args[2:])
		args := terminateCmd.Args()
		if len(args) < 1 {
			fmt.Println("Error: Missing <AUMID> argument.")
			os.Exit(1)
		}
		client.HandleTerminate(args[0])
	case "list-processes":
		listProcCmd.Parse(os.Args[2:])
		client.HandleListProcesses()
	case "kill":
		killCmd.Parse(os.Args[2:])
		args := killCmd.Args()
		if len(args) < 1 {
			fmt.Println("Error: Missing <PID> argument.")
			os.Exit(1)
		}
		client.HandleKill(args[0])
	case "battery":
		batteryCmd.Parse(os.Args[2:])
		client.HandleBattery()
	case "ls":
		lsCmd.Parse(os.Args[2:])
		args := lsCmd.Args()
		if len(args) < 1 {
			fmt.Println("Error: Missing <location> argument.")
			os.Exit(1)
		}
		client.HandleLs(args[0])
	case "ping":
		pingCmd.Parse(os.Args[2:])
		client.HandlePing()
	case "copy":
		copyCmd.Parse(os.Args[2:])
		args := copyCmd.Args()
		if *pcFlag != "" && len(args) == 1 {
			client.HandleCopyPC(*pcFlag, args[0])
		} else if *mobileFlag != "" && len(args) == 1 {
			client.HandleCopyMobile(*mobileFlag, args[0])
		} else {
			fmt.Println("Error: Invalid syntax. Use wpctrl copy --pc <src> <location> or wpctrl copy --mobile <src> <dest>")
			os.Exit(1)
		}
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println("wpctrl commands:")
	fmt.Println("  wpctrl getinfo")
	fmt.Println("  wpctrl install <file.appx>")
	fmt.Println("  wpctrl list-apps")
	fmt.Println("  wpctrl uninstall <PackageFullName>")
	fmt.Println("  wpctrl launch <AUMID>")
	fmt.Println("  wpctrl terminate <AUMID>")
	fmt.Println("  wpctrl list-processes")
	fmt.Println("  wpctrl kill <PID>")
	fmt.Println("  wpctrl battery")
	fmt.Println("  wpctrl ls <location>")
	fmt.Println("  wpctrl ping")
	fmt.Println("  wpctrl screenshot <output.png>")
	fmt.Println("  wpctrl reboot")
	fmt.Println("  wpctrl copy --pc <filefrompc> <location-onwp>")
	fmt.Println("  wpctrl copy --mobile <filefromwp> <localpath>")
}