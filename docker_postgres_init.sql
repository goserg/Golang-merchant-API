create table sellers (
	seller_id int PRIMARY KEY
);
create table offers (
	offer_id int NOT NULL,
	name text NOT NULL,
	price numeric NOT NULL,
	quantity int NOT NULL,
	available boolean,
	seller_id int REFERENCES sellers ON DELETE CASCADE,
	CONSTRAINT unique_offer UNIQUE (offer_id, seller_id)
);
