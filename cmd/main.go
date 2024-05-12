package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/oggwriter"
)

func createPionRTPPacket(p *discordgo.Packet) *rtp.Packet {
	return &rtp.Packet{
		Header: rtp.Header{
			Version: 2,
			// Taken from Discord voice docs
			PayloadType:    0x78,
			SequenceNumber: p.Sequence,
			Timestamp:      p.Timestamp,
			SSRC:           p.SSRC,
		},
		Payload: p.Opus,
	}
}

var Token string = "TOKEN_HERE"
var channelUsers = make(map[string]int)
var mu sync.Mutex

func main() {
	// Initialize Discord Bot (you need to provide your token here)
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}
	dg.AddHandler(channelCreate)
	dg.AddHandler(voiceStateUpdate)
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	select {}
}

func channelCreate(s *discordgo.Session, c *discordgo.ChannelCreate) {
	if c.Type == discordgo.ChannelTypeGuildVoice {
		fmt.Printf("A new voice channel was created: %s\n", c.ID)
		if err := joinVoiceChannel(s, c.GuildID, c.ID); err != nil {
			fmt.Printf("Error joining voice channel %s: %s\n", c.ID, err)
			return
		}
	}
}

func joinVoiceChannel(s *discordgo.Session, guildID, channelID string) error {
	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, false)
	if err != nil {
		return err
	}
	fmt.Printf("Successfully joined voice channel %s in guild %s\n", channelID, guildID)
	mu.Lock()
	channelUsers[channelID] = 1 // Bot itself is in the channel
	mu.Unlock()
	stop := make(chan struct{})
	// Start a timer to check for empty channel after 5 seconds.
	go checkForEmptyChannel(s, guildID, channelID)
	go handleVoice(vc.OpusRecv, stop)
	return nil
}

func handleVoice(c <-chan *discordgo.Packet, stop <-chan struct{}) {
	files := make(map[uint32]media.Writer)
	print(c)
	for p := range c {
		file, ok := files[p.SSRC]
		if !ok {
			var err error
			file, err = oggwriter.New(fmt.Sprintf("%d.ogg", p.SSRC), 48000, 2)
			if err != nil {
				fmt.Printf("failed to create file %d.ogg, giving up on recording: %v\n", p.SSRC, err)
				return
			}
			files[p.SSRC] = file
		}

		// Construct pion RTP packet from DiscordGo's type.
		rtp := createPionRTPPacket(p)
		err := file.WriteRTP(rtp)
		if err != nil {
			fmt.Printf("failed to write to file files/%d.ogg, giving up on recording: %v\n", p.SSRC, err)
		}

	}

	for _, f := range files {
		f.Close()
	}

}

func convert(fileName string) {

	fmt.Printf("Converting %s to mp3...\n", fileName)

	cmd := "ffmpeg -i " + fileName + " -f mp3 " + fileName + ".mp3"

	out, err := exec.Command("bash", "-c", cmd).Output()

	if err != nil {
		fmt.Println(out, cmd)
		panic(err)
	}

}

func checkForEmptyChannel(s *discordgo.Session, guildID, channelID string) {
	time.Sleep(5 * time.Second)
	mu.Lock()
	if channelUsers[channelID] == 1 {
		fmt.Printf("Leaving empty voice channel %s in guild %s because no other users joined.\n", channelID, guildID)
		leaveVoiceChannel(s, guildID, channelID)
	}
	mu.Unlock()
}

func voiceStateUpdate(s *discordgo.Session, vs *discordgo.VoiceStateUpdate) {
	mu.Lock()
	defer mu.Unlock()

	// Handle user leaving a channel
	if vs.BeforeUpdate != nil && vs.BeforeUpdate.ChannelID != "" {
		channelUsers[vs.BeforeUpdate.ChannelID]--
		fmt.Printf("User left channel %s, count now: %d\n", vs.BeforeUpdate.ChannelID, channelUsers[vs.BeforeUpdate.ChannelID])
		if channelUsers[vs.BeforeUpdate.ChannelID] == 1 {
			// Bot is the only user left in the channel
			fmt.Printf("Leaving empty voice channel %s in guild %s because no other users joined.\n", vs.BeforeUpdate.ChannelID, vs.GuildID)
			leaveVoiceChannel(s, vs.GuildID, vs.BeforeUpdate.ChannelID)
		}
	}
	// Handle user joining a channel
	if vs.ChannelID != "" {
		channelUsers[vs.ChannelID]++
		fmt.Printf("User joined channel %s, count now: %d\n", vs.ChannelID, channelUsers[vs.ChannelID])
	}
}

func leaveVoiceChannel(s *discordgo.Session, guildID, channelID string) error {
	fmt.Printf("Attempting to leave voice channel %s in guild %s\n", channelID, guildID)
	vc, ok := s.VoiceConnections[guildID]
	if !ok || vc.ChannelID != channelID {
		fmt.Println("Not connected to the channel.")
		return nil // Not in this channel
	}
	files := oggFiles()
	for _, file := range files {
		go convert(file)
	}
	return vc.Disconnect()
}
func oggFiles() []string {
	fileDivider := "%DV%"

	cmd := "ls | grep -E '\\.ogg(\\n|$)'"

	out, err := exec.Command("bash", "-c", cmd).Output()

	if err != nil {
		panic(err.Error())
	}

	filesBuf := bytes.Buffer{}

	for i := 0; i < len(out); i++ {
		if out[i] == 10 {
			filesBuf.Write([]byte(fileDivider))
			continue
		}

		filesBuf.Write([]byte{out[i]})
	}

	files := strings.Split(filesBuf.String(), fileDivider)

	return files
}
