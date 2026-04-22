# nekobt_notification

Just a simple project to send notifications to a telegram group from nekoBT feed.

## Building from source

Since the tool is written in Go, it should be rather trivial.

1. Ensure that you have Go installed on your system. You can download it from [here](https://golang.org/dl/). At least Go 1.23.1 is required.
2. Clone this repository and switch to the project's root directory
3. Build the project:

```shell
CGO_ENABLED=0 go build -ldflags="-s -w" .
```

And that will produce an `nekobt_notification` binary in the current directory.

If you would rather cross compile, set the `GOOS` and `GOARCH` environment variables accordingly. For example, to build for Windows on a Linux system:

```shell
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" .
```

### Docker

You can deploy the tool using Docker. A [Dockerfile](Dockerfile) and [docker-compose.yml](docker-compose.yml) are provided in the repository.

#### Docker Compose (Recommended)

1. Prepare the configuration and cache files:
   ```shell
   cp config.json.example config.json
   cp cache.json.example cache.json
   ```
2. Edit `config.json` with your settings.
3. Start the container:
   ```shell
   docker compose up -d
   ```

#### Manual Docker Run

To build the image manually:

```shell
docker build -t nekobt_notification:latest .
```

To run it manually:

```shell
docker run -d \
    --name nekobt_notification \
    -v $(pwd)/config.json:/app/config.json \
    -v $(pwd)/cache.json:/app/cache.json \
    nekobt_notification:latest
```

## Usage

Both `config.json` and `cache.json` have to be created, there is example files.

### Config

The config file is `config.json`

- `proxies`: proxy for requesting rss feed, could be needed (ie. `"us": "http://proxy.example.com:8080/"`)
- `chat_id`: ID for the group
- `topic_id`: topic ID if Topics is enabled in the group
- `bot_token`: token for the bot
- `sleep`: time to sleep between each request (in second)
- `rss_feed`: url for the feed ie. https://nekobt.to/api/v1/torrents/search?group_id=7313765950000

### Cache

Items that was already done is in `cache.json`. So can be set where to start. If `cache.json` not mounted with docker, then before ever new run have to update it.
