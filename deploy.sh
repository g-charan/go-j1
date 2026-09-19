set -e
export GOOS=linux GOARCH=arm GOARM=7
go build -o bin/server .
go build -o bin/lb ./lb
adb shell 'ps | grep tmp/ | while read user pid rest; do kill $pid; done'
adb push bin/server bin/lb /data/local/tmp/
adb shell "cd /data/local/tmp; chmod 755 server lb; trap '' HUP;
  /data/local/tmp/server -port 8081 -cache -cache-size 200 -cache-ttl 20s < /dev/null > s1.log 2>&1 &
  /data/local/tmp/server -port 8082 -cache -cache-size 200 -cache-ttl 20s < /dev/null > s2.log 2>&1 &
  /data/local/tmp/lb < /dev/null > lb.log 2>&1 &
  sleep 1; cat s1.log s2.log lb.log"
# public tunnel runs on the phone itself; start it only if it is not already up
adb shell 'cd /data/local/tmp; [ -n "$(ps | grep ngrok)" ] || { trap "" HUP; HOME=/data/local/tmp ./ngrok http --config ngrok.yml --url=wispy-uplifted-recycler.ngrok-free.dev --log stdout 8080 < /dev/null > ngrok.log 2>&1 & sleep 5; }; grep -o "url=https://[^ ]*" ngrok.log'
