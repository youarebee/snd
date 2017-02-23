snd: main.go
	go build

/lib/systemd/system/snd.service: snd.service
	cp snd.service /lib/systemd/system/snd.service