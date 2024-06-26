package bot

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func isBotMention(s *discordgo.Session, m *discordgo.Message) bool {
	for _, user := range m.Mentions {
		if user.ID == s.State.User.ID {
			return true
		}
	}
	return false
}

func getReferenceContent(s *discordgo.Session, msg *discordgo.Message) string {
	if ref := msg.MessageReference; ref != nil {
		if mes, err := s.ChannelMessage(ref.ChannelID, ref.MessageID); err == nil {
			return mes.Content
		}
	}
	return ""
}

func replaceMentions(session *discordgo.Session, guildID string, message string) string {
	message = strings.ReplaceAll(message, "@everyone", "everyone")
	message = strings.ReplaceAll(message, "@here", "here")

	patterns := map[*regexp.Regexp]func(string) string{

		regexp.MustCompile(`<@!?(\d+)>`): func(id string) string {
			member, err := session.GuildMember(guildID, id)
			if err != nil {
				return "@" + id
			}
			if member.Nick != "" {
				return "@" + member.Nick
			}
			return "@" + member.User.Username
		},

		regexp.MustCompile(`<@&(\d+)>`): func(id string) string {
			role, err := session.State.Role(guildID, id)
			if err != nil {
				return "@" + id
			}
			return "@" + role.Name
		},

		regexp.MustCompile(`<#(\d+)>`): func(id string) string {
			channel, err := session.State.Channel(id)
			if err != nil {
				return "#" + id
			}
			return "#" + channel.Name
		},
	}

	for pattern, handler := range patterns {
		message = pattern.ReplaceAllStringFunc(message, func(match string) string {
			id := pattern.FindStringSubmatch(match)[1]
			return handler(id)
		})
	}

	return message
}

func GenerateReply(s *discordgo.Session, m *discordgo.MessageCreate, bot *Bot) {
	if m.Author.ID == s.State.User.ID || !isBotMention(s, m.Message) {
		return
	}

	if id, ok := bot.GuildSettings[m.GuildID]; ok {
		if id != m.GuildID {
			s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("I can't answer you here, go to <#%s>.", id), m.Reference())
			return
		}
	}

	if !bot.Config.ApiConfig.UpdateUserCounter(m.Author.ID) {
		cooldown := bot.Config.ApiConfig.Users[m.Author.ID].EndOfCooldown.Unix()
		timestamp := fmt.Sprintf("You have reached your request limit, the next request can be made <t:%d:R>.", cooldown)
		s.ChannelMessageSendReply(m.ChannelID, timestamp, m.Reference())
		return
	}

	m.Content = strings.ReplaceAll(m.Content, "<@"+s.State.User.ID+">", "<@Your mention>")

	res, err := bot.Config.ApiConfig.Generate(
		m.Content, getReferenceContent(s, m.Message), m.ChannelID,
	)
	if err != nil {
		log.Println("Error generating response: ", err)

		s.ChannelMessageSendReply(m.ChannelID,
			"Oh, something happened, write me again😊",
			m.Reference(),
		)
		return
	}

	bot.Config.ApiConfig.AddToHistory(m.ChannelID, res.Context)

	if len(res.Response) > 2000 {
		res.Response = res.Response[:2000]
	}

	response := fmt.Sprintf("%s\n\nResponse duration: %.2fs",
		replaceMentions(s, m.GuildID, res.Response),
		time.Duration(res.TotalDuration).Seconds(),
	)

	s.ChannelMessageSendReply(m.ChannelID, response, m.Reference())
}
