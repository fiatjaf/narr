## Compilation

Install `Go >= 1.17` and `GCC`. Get the source code:

    git clone https://github.com/fiatjaf/narr.git

Then run one of the corresponding commands:

    # create an executable for the host os
    make build_macos    # -> _output/macos/narr.app
    make build_linux    # -> _output/linux/narr
    make build_windows  # -> _output/windows/narr.exe

    # host-specific cli version (no gui)
    make build_default  # -> _output/narr

    # ... or start a dev server locally
    make serve          # starts a server at http://localhost:7070

    # ... or build a docker image
    docker build -t narr -f etc/dockerfile .

## ARM compilation

The instructions below are to cross-compile *narr* to `Linux/ARM*`.

Build:

    docker build -t narr.arm -f etc/dockerfile.arm .

Test:

    # inside host
    docker run -it --rm narr.arm

    # then, inside container
    cd /root/out
    qemu-aarch64 -L /usr/aarch64-linux-gnu/ narr.arm64

Extract files from images:

    CID=$(docker create narr.arm)
    docker cp -a "$CID:/root/out" .
    docker rm "$CID"
