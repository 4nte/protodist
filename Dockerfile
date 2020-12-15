FROM alpine/git:v2.26.2

COPY protodist /
ENTRYPOINT ["/protodist"]