git pull
docker build -t tw-notification .
docker run -d -p 6996:6996 tw-notification