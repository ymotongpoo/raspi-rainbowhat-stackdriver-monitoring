TARGET = bmp280
DEPS   = main.go

.PHONY: clean rebuild deploy

build: $(DEPS)
	GOOS=linux GOARCH=arm GOARM=6 go build -o $(TARGET) 

clean:
	go clean
	rm -rf $(TARGET)

rebuild: clean build

deploy: rebuild
	scp $(TARGET) pi@pimoroni2.local:~