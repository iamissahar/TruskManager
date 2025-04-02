net:
	docker network create trumpnetwork

build.app:
	docker build -f Dockerfile.app -t trump-app .

build.db:
	docker build -f Dockerfile.db -t trump-postgres .

run.app:
	docker run --name trump-app-container \
	--network trumpnetwork \
	-p 8080:8080 \
	-e host_db=trump-postgres-container \
	-e port_db=5432 \
	-e user_db=postgres \
	-e password_db=postgres \
	-e dbname_db=postgres \
	-e sslmode_db=disable \
	trump-app

run.db:
	docker run --rm --name trump-postgres-container \
	--network trumpnetwork \
	-d \
	-e POSTGRES_USER=postgres \
	-e POSTGRES_PASSWORD=postgres \
	-e POSTGRES_DB=postgres \
	-v ./pgdata:/var/lib/postgresql/data \
	-v ./postgres/create.sql:/docker-entrypoint-initdb.d/c \
	-p 3333:5432 \
	trump-postgres

run.db.outnet:
	docker run --name trump-postgres-container \
	-d \
	-e POSTGRES_USER=postgres \
	-e POSTGRES_PASSWORD=postgres \
	-e POSTGRES_DB=postgres \
	-v ./pgdata:/var/lib/postgresql/data \
	-v ./postgres/create.sql:/docker-entrypoint-initdb.d/c \
	-p 3333:5432 \
	trump-postgres

del.db:
	docker stop trump-postgres-container
	# docker rm trump-postgres-container
	sudo rm -rf ./pgdata

del.app:
	docker stop trump-app-container
	docker rm trump-app-container

reboot:
	docker stop trump-app-container
	docker rm trump-app-container
	docker build -f Dockerfile.app -t trump-app .
	docker run --name trump-app-container \
	--network trumpnetwork \
	-p 8080:8080 \
	-e host_db=trump-postgres-container \
	-e port_db=5432 \
	-e user_db=postgres \
	-e password_db=postgres \
	-e dbname_db=postgres \
	-e sslmode_db=disable \
	trump-app
