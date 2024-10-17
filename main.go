package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/osm/quake/common/args"
	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/clc"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/stringcmd"
	"github.com/osm/quake/packet/command/stufftext"
	"github.com/osm/quake/packet/svc"
	"github.com/osm/quake/proxy"
)

var version string = "1.0.0"

type app struct {
	aliases        []string
	allowDownloads bool
	commands       []string
	logger         *log.Logger
	verbose        bool
}

func main() {
	addrPort := flag.String("listen-addr", "localhost:27500", "listen address")
	allowDownloads := flag.Bool("allow-downloads", false, "toggle if client-initiated downloads should be allowed")
	configFile := flag.String("config-file", "qwstfw.cfg", "path to the qwstfw configuration file")
	verbose := flag.Bool("verbose", false, "enable verbose logging")
	flag.Parse()

	logger := log.New(os.Stderr, "INFO: ", log.Ldate|log.Ltime)
	logger.Printf("QuakeWorld stufftext firewall v%v\n", version)

	app := &app{
		allowDownloads: *allowDownloads,
		logger:         logger,
		verbose:        *verbose,
	}

	if err := app.parseConfigFile(*configFile); err != nil {
		logger.Fatalf("%v\n", err)
	}

	prx := proxy.New(proxy.WithLogger(logger))

	if !app.allowDownloads {
		logger.Printf("client-initiated downloads: disabled\n")
		prx.HandleFunc(proxy.CLC, app.clcHandler)
	} else {
		logger.Printf("client-initiated downloads: enabled\n")
	}
	logger.Printf("number of allowed commands: %d", len(app.commands))
	logger.Printf("number of aliases to inject: %d", len(app.aliases))

	prx.HandleFunc(proxy.SVC, app.svcHandler)

	logger.Printf("listening on %s", *addrPort)
	if err := prx.Serve(*addrPort); err != nil {
		logger.Fatalf("unable to serve, %v", err)
	}
}

func (a *app) parseConfigFile(configFile string) error {
	file, err := os.Open(filepath.Base(configFile))
	if err != nil {
		file, err = os.Open(configFile)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %v", configFile, err)
		}
	}
	defer file.Close()

	var typ string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		l := strings.TrimSpace(scanner.Text())
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}

		if l == "[qwstfw]" {
			typ = "qwstfw"
			continue
		} else if l == "[aliases]" {
			typ = "aliases"
			continue
		} else if l == "[commands]" {
			typ = "commands"
			continue
		}

		if typ == "qwstfw" && strings.HasPrefix(l, "allow_downloads") {
			t := strings.Split(l, "=")
			if strings.TrimSpace(strings.ToLower(t[len(t)-1])) == "true" {
				a.allowDownloads = true
			}
		}

		if typ == "aliases" {
			a.aliases = append(a.aliases, l)
		}

		if typ == "commands" {
			a.commands = append(a.commands, l)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file %s: %v", configFile, err)
	}

	return nil
}

func (a *app) clcHandler(c *proxy.Client, packet packet.Packet) {
	gameData, ok := packet.(*clc.GameData)
	if !ok {
		return
	}

	for i := 0; i < len(gameData.Commands); {
		stringCmd, ok := gameData.Commands[i].(*stringcmd.Command)
		if !ok {
			i++
			continue
		}

		if strings.HasPrefix(stringCmd.String, "download ") {
			a.logger.Printf("blocking %s\n", stringCmd.String)
			c.SVCInject.Enqueue(
				&stufftext.Command{
					String: "echo The download has been blocked by qwstfw\n",
				},
				&stufftext.Command{
					String: "disconnect\n",
				},
			)
			gameData.Commands = append(
				gameData.Commands[:i],
				gameData.Commands[i+1:]...,
			)
			continue
		}

		i++
	}
}

func (a *app) svcHandler(c *proxy.Client, packet packet.Packet) {
	gameData, ok := packet.(*svc.GameData)
	if !ok {
		return
	}

	for i := 0; i < len(gameData.Commands); {
		stufftextCmd, ok := gameData.Commands[i].(*stufftext.Command)
		if !ok {
			i++
			continue
		}

		var cmds []string
		for _, arg := range args.Parse(stufftextCmd.String) {
			cmd := arg.Cmd
			if len(arg.Args) > 0 {
				cmd = fmt.Sprintf("%s %s", cmd, strings.Join(arg.Args, " "))
			}

			if strings.HasPrefix(cmd, "on_enter") ||
				strings.HasPrefix(cmd, "on_spec_enter") {
				c.SVCInject.Enqueue(a.aliasCommands()...)
			}

			if a.isAllowedCommand(cmd) {
				if a.verbose {
					a.logger.Printf("allowing command: %v\n", cmd)
				}
				cmds = append(cmds, cmd)
			} else if a.verbose {
				a.logger.Printf("blocking command: %v\n", cmd)
			}
		}

		if len(cmds) == 0 {
			gameData.Commands = append(
				gameData.Commands[:i],
				gameData.Commands[i+1:]...,
			)
			continue
		}

		stufftextCmd.String = fmt.Sprintf("%s\n", strings.Join(cmds, ";"))
		i++
	}
}

func (a *app) aliasCommands() []command.Command {
	var cmds []command.Command

	for _, a := range a.aliases {
		cmds = append(cmds, &stufftext.Command{String: fmt.Sprintf("%s\n", a)})
	}

	return cmds
}

func (a *app) isAllowedCommand(cmd string) bool {
	var allowed bool

	for i := 0; i < len(a.commands); i++ {
		if strings.HasPrefix(cmd, a.commands[i]) {
			allowed = true
			break
		}
	}

	return allowed
}
