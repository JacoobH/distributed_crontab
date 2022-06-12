FROM golang
RUN mkdir -p /home/work/crontab/worker
WORKDIR /home/work/crontab/worker
COPY main main
COPY worker.json worker.json
RUN chmod +x main
ENTRYPOINT ["./main"]