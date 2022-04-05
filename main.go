package main

import (
	"os"
	"os/signal"
	"syscall"
	"log"
	"fmt"
	"time"
	"strings"
	"errors"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/pelletier/go-toml/v2"
	"github.com/gomarkdown/markdown"
)

var cfg struct {
	Token	string
	GuildID	discord.GuildID
	ErrorChannelID discord.ChannelID
}

var client 		*api.Client

var filter = strings.NewReplacer(
	"```", "",
	"\n", "<br>",
)

func main() {
	// Get the config file
	configFile, err := os.ReadFile("config.toml")
	if(err != nil) {log.Fatalln(err)}

	// Unmarshal the contents into the global config object
	err = toml.Unmarshal([]byte(configFile),&cfg)
	if(err != nil) {log.Fatalln(err)}

	// Initialize the client
	client = api.NewClient("Bot "+cfg.Token)
	// Start a ticker
	ticker := time.NewTicker(30 * time.Second)
	// Update the texts.
	update()
	// Set up signals for terminating the program
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs,syscall.SIGINT, syscall.SIGTERM)
    // Start waiting every 30 seconds
    // (todo: make this shit a command)
    for {
	select {
        case <- ticker.C:
            update()
    	case <- sigs:
        	ticker.Stop()
        	return
        }
    }
}

func update() {
	channels, err := client.Channels(cfg.GuildID)
	if(err != nil) {
		fmt.Println(err)
		return
	}
	// for each channel we're seeing...
	for _, ch := range channels {
		// channels with the exclude option aren't read.
		if(ch.Topic == "EXCLUDE") {
			return
		}
		if err := updateChannel(ch); err != nil {
			_, err = client.SendMessage(cfg.ErrorChannelID, 
				fmt.Sprintf("**Error updating %s**: %s", ch.Mention(), err))
			if err != nil {
				log.Println("sending error message: ", err)
			}
		}
	}
}

func updateChannel(ch discord.Channel) error {
	topic := ch.Topic
	options := strings.Split(topic,";")
	if(len(options) < 0) {
		return errors.New("At least one config option is needed.")
	}
	var header, footer []byte
	var err error
	if(len(options) > 1) {header, err = os.ReadFile(options[1])}
	if(len(options) > 2) {footer, err = os.ReadFile(options[2])}
	if(err != nil) {return err}
	// get the messages in it.
	messages, err := client.Messages(ch.ID, 1)
	if(err != nil) {return err}
	// for each of the messages...
	for _, msg := range messages {
		// pretty it up
		content := filter.Replace(msg.Content)
		md := markdown.ToHTML([]byte(content), nil, nil)
		// open the file to write to
		file, err := os.Open(options[0])
		if(err != nil) {return err}
		defer file.Close()
		file.Write(header)
		file.Write(md)
		file.Write(footer)
		if(err != nil) {return err}
	}
	return nil
}
