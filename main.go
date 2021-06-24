package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"gopkg.in/fsnotify.v1"
	"gopkg.in/yaml.v3"
)

type (
	Config struct {
		// Commands are startup commands.
		Commands []Command `yaml:"commands"`
		// Watchers watcher configurations.
		Watchers []*Watcher `yaml:"watchers"`
	}
	Watcher struct {
		Enable   bool      `yaml:"enable"`
		Targets  []string  `yaml:"targets"`
		Commands []Command `yaml:"commands"`
	}
	Command struct {
		Timeout time.Duration `yaml:"timeout"`
		Dir     string        `yaml:"dir"`
		Command string        `yaml:"command"`
		Args    []string      `yaml:"args"`
	}
)

func main() {
	conf := flag.String("conf", "watch.yml", "Watch configurations")
	flag.Parse()
	Watch(ReadConfig(*conf))
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan
}

func ReadConfig(p string) *Config {
	f, err := os.ReadFile(p)
	if err != nil {
		log.Panic(err)
	}
	var conf Config
	if err := yaml.Unmarshal(f, &conf); err != nil {
		log.Panic(err)
	}
	return &conf
}

func Watch(config *Config) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("ERROR: start file watcher, err: %v\n", err)
		return err
	}
	log.Println("running startup commands...")
	go func() {
		for _, cmd := range config.Commands {
			cmd.Run()
		}
	}()
	for _, conf := range config.Watchers {
		if !conf.Enable {
			continue
		}
		conf := conf
		go func() {
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					if event.Op&fsnotify.Write == fsnotify.Write {
						log.Printf("INFO: %s changed\n", event.Name)
						for _, cmd := range conf.Commands {
							cmd.Run()
						}
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					log.Println("ERROR: ", err)
				}
			}
		}()
		for _, p := range conf.Targets {
			if err = watcher.Add(p); err != nil {
				log.Printf("ERROR: ailed to watch %q, err: %v\n", p, err)
			}
			log.Printf("INFO: watching: %s\n", p)
		}
	}
	return nil
}

func (cmd *Command) String() string {
	return fmt.Sprintf("%s %s", cmd.Command, strings.Join(cmd.Args, " "))
}

func (cmd *Command) Run() {
	log.Printf("INFO: running: %s\n", cmd.String())
	defer func() {
		if err := recover(); err != nil {
			log.Printf("ERROR: hot reload failed, err: %v\n", err)
		}
	}()
	switch {
	case (cmd.Command == "sleep" && len(cmd.Args) == 1):
		duration, err := time.ParseDuration(cmd.Args[0])
		if err != nil {
			log.Printf("ERROR: cmd %s, err: %s\n", cmd.String(), err)
			break
		}
		time.Sleep(duration)
	default:
		timeout := cmd.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		c := exec.CommandContext(ctx, cmd.Command, cmd.Args...)
		c.Dir = cmd.Dir
		c.Stdout = os.Stdout
		if err := c.Run(); err != nil {
			log.Printf("ERROR: cmd %s, err: %s\n", cmd.String(), err)
		}
	}
}
