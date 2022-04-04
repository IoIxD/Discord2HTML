package main

import (
	"os"
	"log"
	"fmt"
	"time"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/pelletier/go-toml/v2"
	"github.com/gomarkdown/markdown"
)

type Config struct {
	Token	string
	GuildID	discord.GuildID
	ErrorChannelID discord.ChannelID
}

var client 		*api.Client
var ticker 		*time.Ticker
var cfg 		Config


func main() {
	go discordInit() // initialize everything
	for {}
}

func discordInit() {
	// Get the config file
	configFile, err := ReadFile("config.toml")
	if(err != nil) {log.Fatalln(err)}

	// Unmarshal the contents into the global config object
	err = toml.Unmarshal([]byte(configFile),&cfg)
	if(err != nil) {log.Fatalln(err)}

	// Initialize the client
	client = api.NewClient("Bot "+cfg.Token)
	// Start a ticker
	ticker = time.NewTicker(30 * time.Second)
	// Update the texts.
	discordUpdate()
	go discordUpdateTick() // start the update process
}

func discordUpdateTick() {
    quit := make(chan struct{})
    for {
	select {
        case <- ticker.C:
            discordUpdate()
    	case <- quit:
        	ticker.Stop()
        	return
        }
    }
}

func discordUpdate() {
	channels, err := client.Channels(cfg.GuildID)
	if(err != nil) {
		fmt.Println(err)
	} else {
		// for each channel we're watching...
		for i := 0; i < len(channels); i++ {
			// get options based on it's topic
			topic := channels[i].Topic
			if(topic != "EXCLUDE") { // channels with the exclude option aren't read.
				name := channels[i].Name
				fmt.Println("Reading "+name)
				options := strings.Split(topic,";")
				if(len(options) < 0) {
					client.SendMessage(cfg.ErrorChannelID, "**Error updating "+name+"**: At least one config option is needed.")
				} else {
					header := ""
					footer := ""
					if(len(options) > 1) {
						header, err = ReadFile(options[1])
						if(err != nil) {client.SendMessage(cfg.ErrorChannelID, "**Error updating "+name+"**: "+err.Error())}
					}
					if(len(options) > 2) {
						footer, err = ReadFile(options[2])
						if(err != nil) {client.SendMessage(cfg.ErrorChannelID, "**Error updating "+name+"**: "+err.Error())}
					}
					// get the messages in it.
					messages, err := client.Messages(channels[i].ID, 1)
					if(err != nil) {client.SendMessage(cfg.ErrorChannelID, "**Error updating "+name+"**: "+err.Error())}
					// for each of the messages...
					for n := 0; n < len(messages); n++ {
						// get that message
						message := messages[n].Content
						// pretty up the message to prepare to serve it
						message = strings.Replace(message,"```","",2)
						message = strings.Replace(message,"\n","<br>",67676)
						message = string(markdown.ToHTML([]byte(message), nil, nil))
						// write that message to a file
						err = os.WriteFile(options[0],[]byte(header+message+footer),0666)
						if(err != nil) {client.SendMessage(cfg.ErrorChannelID, "**Error updating "+name+"**: "+err.Error())}
					}
				}
			}
		}
	}
}

func ReadFile(filename string) (_ string, err error) {
	// Open the config file and save its contents
	file, err := os.Open(filename)
	if(err != nil) {return "", err}
	stat, err := file.Stat()
	if(err != nil) {return "", err}
	contents := make([]byte,stat.Size())
	_, err = file.Read(contents);
	if(err != nil) {return "", err}
	return string(contents), nil
}
