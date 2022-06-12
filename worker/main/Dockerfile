FROM golang:1.18
RUN mkdir /home/work/crontab/worker
WORKDIR /home/work/crontab/worker
COPY main main
COPY worker.json worker.json
RUN chmod +x main
EXPOSE 8080
ENTRYPOINT ["./main"]