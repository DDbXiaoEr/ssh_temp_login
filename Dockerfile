FROM debian:bullseye-v1.4.1

COPY build/ssh-sign-linux /usr/local/bin/ssh-sign
COPY docker/config.yaml /etc/ssh-sign/config.yaml

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/ssh-sign"]
CMD ["-config", "/etc/ssh-sign/config.yaml"]
