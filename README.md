# Golang merchant API

## Запуск

    docker-compose up

## Документация по API


### Загрузка данных по товарам в базу данных

#### Запрос

**POST** /offers

Request Body schema: application/json

*seller_id* (int, required): ID продавца в нашей системе

*url* (string, required): Адрес xlsx файла

*async* (boolean, default=false): Выполнение запроса в асинхронном режиме

#### Ответ (асинхронный режим)

Response Schema: application/json

	{
		"task_id": integer
	}
    
#### Ответ (синхронный режим)

См. ответ на GET запрос /info

#### Коды ответов

200: Успешная обработка запроса

400: Неверный запрос

503: API временно недоступен


### Информация по задаче

#### Запрос

**GET** /info

Request Body schema: application/json

*task_id* (int, required): ID задачи

#### Ответ

Response Schema: application/json

	{
		"task_id":		integer,
		"status":		string,
		"elapsed_time":		string,
		"lines_parsed":		integer,
		"new_offers":		integer,
		"updated_offers":	integer,
		"errors":		integer
	}
    
    
#### Коды ответов

200: Успешная обработка запроса

400: Неверный запрос

404: Задача не найдена

503: API временно недоступен


### Поиск по базе данных

#### Запрос

**GET** /offers

Request Body schema: application/json

*seller_id* (int, required): ID продавца в нашей системе

*offer_id* (int, required): ID товара в системе продавца

*name_search* (string, required): Строка поиска по имени товара

#### Ответ

Response Schema: application/json

	[
		{
			"offer_id":	integer,
			"name":		string,
			"price":	float,
			"quantity":	int,
			"available":	boolean,
			"seller_id":	integer
		},
		...
	]


