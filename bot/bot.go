package bot

import (
	"fmt"
	"ollama-discord/config"
	"ollama-discord/events"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	Session *discordgo.Session
	Config  *config.Config
}

type command struct {
	data    *discordgo.ApplicationCommand
	execute func(*discordgo.Session, *discordgo.InteractionCreate)
}

var commands = map[string]command{
	"clear": {
		data: &discordgo.ApplicationCommand{
			Name:        "clear",
			Description: "Clears context history in this channel",
		},
		execute: func(s *discordgo.Session, i *discordgo.InteractionCreate) {

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Hey there! Congratulations, you just executed your first slash command",
				},
			})
		},
	},
}

func (bot *Bot) RegisterSlashCommands(s *discordgo.Session) (string, error) {
	for _, v := range commands {
		_, err := s.ApplicationCommandCreate(
			s.State.User.ID, "", v.data)
		if err != nil {
			return v.data.Name, err
		}
	}
	return "", nil
}

func NewBot(session *discordgo.Session, config *config.Config) *Bot {
	return &Bot{
		Session: session,
		Config:  config,
	}
}

func (bot *Bot) RegisterHandlers() {
	bot.Session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Printf("Loging as %s#%s\n", r.User.Username, r.User.Discriminator)
		s.UpdateGameStatus(0, "Chat with AI")
	})

	bot.Session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		events.GenerateReply(s, m, &bot.Config.ApiConfig)
		events.Ping(s, m)
	})

	bot.Session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if c, ok := commands[i.ApplicationCommandData().Name]; ok {
			c.execute(s, i)
		}
	})
}
