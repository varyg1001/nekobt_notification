#!/bin/bash

IMAGE_NAME="nekobt_notification:latest"
CONTAINER_NAME="nekobt_notification"
CONFIG_PATH="."
CACHE_FILE="cache.json"
CACHE_TEMPLATE="cache.json.example"

show_help() {
    echo "Usage: ./docker_run.sh [OPTIONS]"
    echo "Options:"
    echo "  -h, --help       Show this help message"
    echo "  -b, --build      Build the Docker image"
    echo "  -r, --rebuild    Force rebuild the Docker image"
    echo "  -s, --stop       Stop running container"
    echo "  -c, --clean      Remove container and image"
}

build_image() {
    echo "Building Docker image..."
    sudo docker build -t $IMAGE_NAME .
}

stop_container() {
    if sudo docker ps -q -f name="$CONTAINER_NAME" >/dev/null; then
        echo "Stopping container..."
        sudo docker stop "$CONTAINER_NAME"
    fi
}

remove_container() {
    if sudo docker ps -aq -f name="$CONTAINER_NAME" >/dev/null; then
        echo "Removing container..."
        sudo docker rm "$CONTAINER_NAME"
    fi
}

remove_image() {
    if sudo docker images -q "$IMAGE_NAME" >/dev/null; then
        echo "Removing Docker image..."
        sudo docker rmi -f "$IMAGE_NAME"
    fi
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)
            show_help
            exit 0
            ;;
        -b|--build)
            BUILD=true
            shift
            ;;
        -r|--rebuild)
            REBUILD=true
            shift
            ;;
        -s|--stop)
            stop_container
            exit 0
            ;;
        -c|--clean)
            stop_container
            remove_container
            remove_image
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

if [ ! -f "$CACHE_FILE" ]; then
  if [ -f "$CACHE_TEMPLATE" ]; then
    cp "$CACHE_TEMPLATE" "$CACHE_FILE"
  else
    echo '{"last_run": []}' > "$CACHE_FILE"
  fi
fi

if [ "$REBUILD" = true ]; then
    stop_container
    remove_container
    remove_image
    build_image
fi

if [ "$BUILD" = true ]; then
    build_image
fi

if ! sudo docker images -q "$IMAGE_NAME" >/dev/null; then
    echo "Image not found. Building..."
    build_image
fi

stop_container
remove_container

echo "Starting container..."
sudo docker run -d \
    --name "$CONTAINER_NAME" \
    -v "$(pwd)/cache.json:/app/$CACHE_FILE" \
    "$IMAGE_NAME"

echo "Container started. To view logs, use: docker logs $CONTAINER_NAME"
