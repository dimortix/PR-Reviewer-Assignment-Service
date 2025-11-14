CREATE TABLE IF NOT EXISTS teams (
    team_name VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS users (
    user_id VARCHAR(255) PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    team_name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (team_name) REFERENCES teams(team_name) ON DELETE CASCADE
);

-- индексы для быстрого поиска по команде и статусу активности
CREATE INDEX idx_users_team_name ON users(team_name);
CREATE INDEX idx_users_is_active ON users(is_active);

CREATE TABLE IF NOT EXISTS pull_requests (
    pull_request_id VARCHAR(255) PRIMARY KEY,
    pull_request_name VARCHAR(255) NOT NULL,
    author_id VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('OPEN', 'MERGED')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    merged_at TIMESTAMP WITH TIME ZONE,
    FOREIGN KEY (author_id) REFERENCES users(user_id)
);

-- индексы для фильтрации по статусу и автору
CREATE INDEX idx_pull_requests_status ON pull_requests(status);
CREATE INDEX idx_pull_requests_author ON pull_requests(author_id);

-- связь многие-ко-многим между PR и ревьюверами
CREATE TABLE IF NOT EXISTS pull_request_reviewers (
    pull_request_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (pull_request_id, user_id),
    FOREIGN KEY (pull_request_id) REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(user_id)
);

-- индекс для получения всех PR по ревьюверу
CREATE INDEX idx_pr_reviewers_user_id ON pull_request_reviewers(user_id);
