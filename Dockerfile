FROM golang AS builder
WORKDIR /app
COPY . .
RUN go get 
RUN go build

FROM alpine
COPY --from=builder /app/sshbox /sshbox
EXPOSE 22
ENTRYPOINT /sshbox