# Discord Audio Recorder and Converter Bot

This project features a Discord bot implemented in Go, capable of joining voice channels, recording audio, and converting the recorded audio from OGG to MP3 format. It utilizes the `discordgo` library to interact with the Discord API and `pion/webrtc` for handling RTP packets.

## Features

- Join automatically created voice channels.
- Record audio in voice channels into OGG format.
- Convert recorded OGG audio files to MP3 format using FFmpeg.
- Leave voice channels if they become empty.
- Track the number of users in each voice channel.

## Requirements

- Go 1.15 or higher
- Discord Bot Token
- FFmpeg installed on the host system
- `discordgo`, `pion/rtp`, and `pion/webrtc/v3` Go libraries

## Setup

1. **Clone the Repository:**
   ```bash
   git clone git@github.com:ud1as/Auto-recording-bot.git
   cd Auto-recording-bot
