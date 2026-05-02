FROM ubuntu:latest

# Install necessary packages
RUN apt-get update && apt-get install -y \
    bash wget vim curl git ca-certificates sqlite3 ffmpeg gcc && \
    rm -rf /var/lib/apt/lists/*

# Install Go using webi and force it into the system PATH
RUN curl -sS https://webi.sh/golang | sh
ENV PATH="/root/.local/bin:/root/.config/envman/PATH.env:$PATH"
# Manually add typical Go bin paths to the ENV so Docker sees them
ENV PATH="/root/go/bin:/usr/local/go/bin:$PATH"

# Install Starship
RUN curl -sS https://starship.rs/install.sh | sh -s -- -y && \
    echo 'eval "$(starship init bash)"' >> ~/.bashrc

# Install Tools (BootDev)
RUN . ~/.config/envman/PATH.env && \
	go install github.com/bootdotdev/bootdev@latest
    # go get github.com/mattn/go-sqlite3@latest && \
    # go get github.com/google/uuid@latest

WORKDIR /tubely

CMD ["/bin/bash"]
