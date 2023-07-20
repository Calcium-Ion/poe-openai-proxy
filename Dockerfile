FROM golang:1.20-alpine

WORKDIR /app
COPY . .

#RUN go build

EXPOSE 3701
CMD [ "/app/poe-openai-proxy" ]