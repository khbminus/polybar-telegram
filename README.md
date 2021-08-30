# Script: polybar-telegram

A go script that shows count of unread telegram messages

It uses telegram client API with gotd. So you should get app ID and app hash
from [Telegram API](https://my.telegram.org/apps)

![polybar-telegram](screenshots/1.png)

## Installation

```shell
go install github.com/Doktorkrab/polybar-telegram
```

## Configuration

polybar-telegram needs some environs:

- TG_APPID / TG_APPHASH - App id and hash from Telegram app config
- PHONE - mobile phone number used for authorization
- AUTH_FILE - path to file that will use to persist login data

After installation, you should run script in terminal with flag -auth to log in into Telegram.

## Script
```ini
[module/telegram]
type = custom/script
exec = $HOME/go/bin/polybar-telegram
interval = 10 
```