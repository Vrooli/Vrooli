FROM python:3.12-slim
WORKDIR /app
COPY kernel /app/kernel
ENTRYPOINT ["python3", "-m", "kernel.host.engine"]
