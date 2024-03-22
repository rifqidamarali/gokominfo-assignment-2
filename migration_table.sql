CREATE TABLE orders (
    order_id SERIAL PRIMARY KEY NOT NULL,
    customer_name VARCHAR(255) NOT NULL, 
    order_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE items(
    item_id serial primary key not null,
    item_code varchar(255) not null, 
    description text,  
    quantity int not null,
    order_id int not null,
    
    constraint fk_item_order_id
        foreign key (order_id)
        references orders(order_id)
);

ALTER TABLE orders
RENAME COLUMN order_at TO ordered_at;