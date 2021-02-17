create table seller (
	id integer,
	PRIMARY KEY (id)
);
create table offer (
	id integer,
	name text NOT NULL,
	price real NOT NULL,
	quantity integer NOT NULL,
	available boolean,
	seller_id integer REFERENCES seller ON DELETE CASCADE,
	CONSTRAINT offer_seller_id UNIQUE (id, seller_id)
);
create table task_log (
	id BIGSERIAL,
	url char(2000),
	seller_id integer REFERENCES seller ON DELETE CASCADE,
	status text,
	elapsed_time text,
	lines_parsed integer,
	new_offers integer,
	updated_offers integer,
	errors integer,
	PRIMARY KEY (id)
);
