CREATE TABLE daily_entries
(
    id      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id INT NOT NULL ,
    text VARCHAR(500) NOT NULL ,
    date DATE NOT NULL ,
    rating INTEGER NOT NULL CHECK ( rating >= 1 AND rating <= 10 ),
    FOREIGN KEY (user_id)
    REFERENCES users(id)
    CONSTRAINT unique_user_date UNIQUE (user_id, date)
);