CREATE TABLE IF NOT EXISTS users (
	id              INTEGER         AUTO_INCREMENT,
	name            VARCHAR(256)    UNIQUE NOT NULL,
	password_hash   CHAR(64)        NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS posts (
    id          INTEGER     AUTO_INCREMENT,
    user_id     INTEGER(64) NOT NULL,
    title       VARCHAR(64) NOT NULL,
    body        VARCHAR(64) NOT NULL,
    available   BOOLEAN NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);