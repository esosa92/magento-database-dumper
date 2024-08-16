#!/usr/bin/env bash
	
run: ## Start the containers
	@docker compose up -d

stop: ## Stop the containers
	@docker compose stop
connect: ##Connect to the container
	@docker exec -it magento_db_dumper bash
