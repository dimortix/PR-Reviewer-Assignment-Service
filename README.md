# PR Reviewer Assignment Service

Микросервис для автоматического назначения ревьюеров на Pull Request'ы. Поддерживает управление командами разработчиков и гибкое переназначение ревьюверов.

## Основной функционал

Сервис предоставляет следующие возможности:
- Автоматическое назначение до 2 ревьюеров из команды автора PR
- Переназначение ревьюверов с учетом команды заменяемого участника
- Управление командами и списком участников
- Управление активностью пользователей
- Идемпотентная операция слияния PR
- Статистика по пользователям (количество PR и ревью)
- Graceful shutdown с корректным завершением соединений
- Структурированное логирование с настраиваемыми уровнями
- Connection pooling для оптимальной работы с PostgreSQL
- Health check endpoint для проверки работоспособности сервиса
- Конфигурация через переменные окружения
- Транзакционность всех критичных операций

## Системные требования

Для запуска сервиса необходимо:
- Docker версии 20.10 или выше
- Docker Compose V2
- Все дополнительные средства тестирования(golangci и тд) должны быть установлены на вашей системе
## Запуск сервиса

Выполните команду для запуска всех компонентов:

```make
make docker-up
```
или 

```bash
docker-compose up --build
```

После запуска сервис будет доступен по адресу http://localhost:8080

Процесс запуска автоматически выполнит следующие шаги:
1. Запуск PostgreSQL 17
2. Применение миграций базы данных
3. Запуск приложения на порту 8080

## Описание API

Для тестирования используйте файл [examples.http](examples.http) с расширением REST Client для VSCode, или команды ниже.

### Управление командами

Создание новой команды с участниками. Если команда с таким именем уже существует, возвращается ошибка TEAM_EXISTS.

Bash/Linux/Mac:
```bash
curl -X POST http://localhost:8080/team/add \
  -H "Content-Type: application/json" \
  -d '{
    "team_name": "backend",
    "members": [
      {
        "user_id": "u1",
        "username": "Alexey",
        "is_active": true
      },
      {
        "user_id": "u2",
        "username": "Dmitry",
        "is_active": true
      }
    ]
  }'
```

Windows PowerShell:
```powershell
$body = @'
{
  "team_name": "backend",
  "members": [
    {
      "user_id": "u1",
      "username": "Alexey",
      "is_active": true
    },
    {
      "user_id": "u2",
      "username": "Dmitry",
      "is_active": true
    }
  ]
}
'@
Invoke-RestMethod -Uri "http://localhost:8080/team/add" -Method POST -ContentType "application/json" -Body $body
```

Обновление существующей команды. Добавляет новых участников или обновляет информацию о существующих. Если команда не найдена, возвращается ошибка NOT_FOUND.

Bash/Linux/Mac:
```bash
curl -X POST http://localhost:8080/team/update \
  -H "Content-Type: application/json" \
  -d '{
    "team_name": "backend",
    "members": [
      {
        "user_id": "u1",
        "username": "Alexey",
        "is_active": true
      },
      {
        "user_id": "u2",
        "username": "Dmitry",
        "is_active": true
      },
      {
        "user_id": "u3",
        "username": "Igor",
        "is_active": true
      }
    ]
  }'
```

Windows PowerShell:
```powershell
$body = @'
{
  "team_name": "backend",
  "members": [
    {
      "user_id": "u1",
      "username": "Alexey",
      "is_active": true
    },
    {
      "user_id": "u2",
      "username": "Dmitry",
      "is_active": true
    },
    {
      "user_id": "u3",
      "username": "Igor",
      "is_active": true
    }
  ]
}
'@
Invoke-RestMethod -Uri "http://localhost:8080/team/update" -Method POST -ContentType "application/json" -Body $body
```

Получение информации о команде с полным списком участников.

Bash/Linux/Mac:
```bash
curl http://localhost:8080/team/get?team_name=backend
```

Windows PowerShell:
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/team/get?team_name=backend" -Method GET
```

### Управление пользователями

Изменение статуса активности пользователя. Неактивные пользователи не назначаются на ревью.

Bash/Linux/Mac:
```bash
curl -X POST http://localhost:8080/users/setIsActive \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "u2",
    "is_active": false
  }'
```

Windows PowerShell:
```powershell
$body = @'
{
  "user_id": "u2",
  "is_active": false
}
'@
Invoke-RestMethod -Uri "http://localhost:8080/users/setIsActive" -Method POST -ContentType "application/json" -Body $body
```

Получение списка PR, где пользователь назначен ревьювером.

Bash/Linux/Mac:
```bash
curl http://localhost:8080/users/getReview?user_id=u1
```

Windows PowerShell:
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/users/getReview?user_id=u1" -Method GET
```

### Работа с Pull Request

Создание нового PR. Автоматически назначает до 2 активных ревьюверов из команды автора, исключая самого автора.

Bash/Linux/Mac:
```bash
curl -X POST http://localhost:8080/pullRequest/create \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001",
    "pull_request_name": "Add search",
    "author_id": "u1"
  }'
```

Windows PowerShell:
```powershell
$body = @'
{
  "pull_request_id": "pr-1001",
  "pull_request_name": "Add search",
  "author_id": "u1"
}
'@
Invoke-RestMethod -Uri "http://localhost:8080/pullRequest/create" -Method POST -ContentType "application/json" -Body $body
```

Отметка PR как слитого. Операция идемпотентна - повторный вызов для уже слитого PR не вызывает ошибку.

Bash/Linux/Mac:
```bash
curl -X POST http://localhost:8080/pullRequest/merge \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001"
  }'
```

Windows PowerShell:
```powershell
$body = @'
{
  "pull_request_id": "pr-1001"
}
'@
Invoke-RestMethod -Uri "http://localhost:8080/pullRequest/merge" -Method POST -ContentType "application/json" -Body $body
```

Переназначение конкретного ревьювера на другого участника из команды заменяемого.

Bash/Linux/Mac:
```bash
curl -X POST http://localhost:8080/pullRequest/reassign \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001",
    "old_user_id": "u2"
  }'
```

Windows PowerShell:
```powershell
$body = @'
{
  "pull_request_id": "pr-1001",
  "old_user_id": "u2"
}
'@
Invoke-RestMethod -Uri "http://localhost:8080/pullRequest/reassign" -Method POST -ContentType "application/json" -Body $body
```

### Статистика

Получение статистики по всем пользователям. Показывает количество созданных PR, назначенных ревью и активных ревью.

Bash/Linux/Mac:
```bash
curl http://localhost:8080/stats
```

Windows PowerShell:
```powershell
# простой вывод
Invoke-RestMethod -Uri "http://localhost:8080/stats" -Method GET

# читабельный форматированный вывод(этот мне больше нравится)
Invoke-RestMethod -Uri "http://localhost:8080/stats" -Method GET | ConvertTo-Json -Depth 10
```

Ожидаемый результат:
```json
{
  "users": [
    {
      "user_id": "u1",
      "username": "Alexey",
      "team_name": "backend",
      "is_active": true,
      "total_prs_authored": 1,
      "total_reviews_assigned": 0,
      "active_reviews": 0
    },
    {
      "user_id": "u3",
      "username": "Igor",
      "team_name": "backend",
      "is_active": true,
      "total_prs_authored": 0,
      "total_reviews_assigned": 1,
      "active_reviews": 0
    },
    {
      "user_id": "u6",
      "username": "Pavel",
      "team_name": "backend",
      "is_active": true,
      "total_prs_authored": 0,
      "total_reviews_assigned": 1,
      "active_reviews": 0
    },
    {
      "user_id": "u2",
      "username": "Dmitry",
      "team_name": "backend",
      "is_active": false,
      "total_prs_authored": 0,
      "total_reviews_assigned": 0,
      "active_reviews": 0
    }
  ]
}
```

HTTP статус: 200 OK

Пояснение: статистика показывает всех пользователей с их активностью:
- total_prs_authored - общее количество PR, созданных пользователем
- total_reviews_assigned - общее количество раз, когда пользователь был назначен ревьювером (учитываются только текущие назначения, переназначенные ревьюверы не учитываются)
- active_reviews - количество открытых PR, где пользователь назначен ревьювером (только PR со статусом OPEN)

Пользователи отсортированы по количеству назначенных ревью (убывание), затем по количеству созданных PR. Это помогает быстро оценить загрузку участников команды.

В данном примере все active_reviews равны 0, потому что PR pr-1001 находится в статусе MERGED. Если бы запросили статистику до слияния (после шага 7), то у u3 и u6 было бы active_reviews=1.

## Пример работы с API

Рассмотрим полный сценарий использования сервиса с пояснением каждого шага(надеюсь не зря расписываю это всё).

### Шаг 1. Создание команды backend

Bash/Linux/Mac:
```bash
curl -X POST http://localhost:8080/team/add \
  -H "Content-Type: application/json" \
  -d '{
    "team_name": "backend",
    "members": [
      {"user_id": "u1", "username": "Alexey", "is_active": true},
      {"user_id": "u2", "username": "Dmitry", "is_active": true},
      {"user_id": "u3", "username": "Igor", "is_active": true}
    ]
  }'
```

Windows PowerShell:
```powershell
$body = @'
{
  "team_name": "backend",
  "members": [
    {"user_id": "u1", "username": "Alexey", "is_active": true},
    {"user_id": "u2", "username": "Dmitry", "is_active": true},
    {"user_id": "u3", "username": "Igor", "is_active": true}
  ]
}
'@
Invoke-RestMethod -Uri "http://localhost:8080/team/add" -Method POST -ContentType "application/json" -Body $body
```

Ожидаемый результат:
```json
{
  "team": {
    "team_name": "backend",
    "members": [
      {"user_id": "u1", "username": "Alexey", "is_active": true},
      {"user_id": "u2", "username": "Dmitry", "is_active": true},
      {"user_id": "u3", "username": "Igor", "is_active": true}
    ]
  }
}
```

HTTP статус: 201 Created

Пояснение: команда backend успешно создана с тремя участниками. Все участники добавлены в таблицу users с привязкой к команде backend. Транзакция гарантирует что либо создастся команда со всеми участниками, либо операция полностью откатится.

### Шаг 2. Попытка создать ту же команду повторно

Bash/Linux/Mac:
```bash
curl -X POST http://localhost:8080/team/add \
  -H "Content-Type: application/json" \
  -d '{
    "team_name": "backend",
    "members": [
      {"user_id": "u4", "username": "Pavel", "is_active": true}
    ]
  }'
```

Windows PowerShell:
```powershell
$body = @'
{
  "team_name": "backend",
  "members": [
    {"user_id": "u4", "username": "Pavel", "is_active": true}
  ]
}
'@
Invoke-RestMethod -Uri "http://localhost:8080/team/add" -Method POST -ContentType "application/json" -Body $body
```

Ожидаемый результат:
```json
{
  "error": {
    "code": "TEAM_EXISTS",
    "message": "TEAM_EXISTS: team_name already exists"
  }
}
```

HTTP статус: 400 Bad Request

Пояснение: команда backend уже существует в базе данных. Согласно спецификации OpenAPI, эндпоинт /team/add должен возвращать ошибку TEAM_EXISTS при попытке создать существующую команду. Это защищает от случайной перезаписи данных команды.

### Шаг 3. Добавление нового участника в существующую команду

Bash/Linux/Mac:
```bash
curl -X POST http://localhost:8080/team/update \
  -H "Content-Type: application/json" \
  -d '{
    "team_name": "backend",
    "members": [
      {"user_id": "u1", "username": "Alexey", "is_active": true},
      {"user_id": "u2", "username": "Dmitry", "is_active": true},
      {"user_id": "u3", "username": "Igor", "is_active": true},
      {"user_id": "u6", "username": "Pavel", "is_active": true}
    ]
  }'
```

Windows PowerShell:
```powershell
$body = @'
{
  "team_name": "backend",
  "members": [
    {"user_id": "u1", "username": "Alexey", "is_active": true},
    {"user_id": "u2", "username": "Dmitry", "is_active": true},
    {"user_id": "u3", "username": "Igor", "is_active": true},
    {"user_id": "u6", "username": "Pavel", "is_active": true}
  ]
}
'@
Invoke-RestMethod -Uri "http://localhost:8080/team/update" -Method POST -ContentType "application/json" -Body $body
```

Ожидаемый результат:
```json
{
  "team": {
    "team_name": "backend",
    "members": [
      {"user_id": "u1", "username": "Alexey", "is_active": true},
      {"user_id": "u2", "username": "Dmitry", "is_active": true},
      {"user_id": "u3", "username": "Igor", "is_active": true},
      {"user_id": "u6", "username": "Pavel", "is_active": true}
    ]
  }
}
```

HTTP статус: 200 OK

Пояснение: эндпоинт /team/update позволяет обновлять состав существующих команд. Новый участник Pavel добавлен в команду. Существующие участники обновляются при необходимости. Этот эндпоинт был добавлен дополнительно для практического удобства работы с командами, при этом строгое соблюдение спецификации OpenAPI для /team/add сохранено.

### Шаг 4. Создание Pull Request

Bash/Linux/Mac:
```bash
curl -X POST http://localhost:8080/pullRequest/create \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001",
    "pull_request_name": "Add search feature",
    "author_id": "u1"
  }'
```

Windows PowerShell:
```powershell
$body = @'
{
  "pull_request_id": "pr-1001",
  "pull_request_name": "Add search feature",
  "author_id": "u1"
}
'@
Invoke-RestMethod -Uri "http://localhost:8080/pullRequest/create" -Method POST -ContentType "application/json" -Body $body
```

Ожидаемый результат:
```json
{
  "pr": {
    "pull_request_id": "pr-1001",
    "pull_request_name": "Add search feature",
    "author_id": "u1",
    "status": "OPEN",
    "assigned_reviewers": ["u2", "u3"],
    "createdAt": "2025-11-14T10:30:00Z",
    "mergedAt": null
  }
}
```

HTTP статус: 201 Created

Пояснение: PR успешно создан со статусом OPEN. Автоматически назначены два ревьювера из команды автора. Алгоритм выбора:
1. Определяется команда автора (u1 из команды backend)
2. Выбираются активные участники команды, исключая автора: u2, u3, u6
3. Случайным образом выбираются до 2 ревьюверов
4. В данном случае назначены u2 и u3

Автор u1 исключается из списка кандидатов, так как нельзя назначать себя ревьювером собственного PR.

### Шаг 5. Просмотр PR, назначенных пользователю

Bash/Linux/Mac:
```bash
curl http://localhost:8080/users/getReview?user_id=u2
```

Windows PowerShell:
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/users/getReview?user_id=u2" -Method GET
```

Ожидаемый результат:
```json
{
  "user_id": "u2",
  "pull_requests": [
    {
      "pull_request_id": "pr-1001",
      "pull_request_name": "Add search feature",
      "author_id": "u1",
      "status": "OPEN"
    }
  ]
}
```

HTTP статус: 200 OK

Пояснение: пользователь u2 видит все PR, где он назначен ревьювером. В данном случае это только pr-1001. Этот эндпоинт полезен для отслеживания рабочей нагрузки каждого участника команды.

### Шаг 6. Деактивация пользователя

Bash/Linux/Mac:
```bash
curl -X POST http://localhost:8080/users/setIsActive \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "u2",
    "is_active": false
  }'
```

Windows PowerShell:
```powershell
$body = @'
{
  "user_id": "u2",
  "is_active": false
}
'@
Invoke-RestMethod -Uri "http://localhost:8080/users/setIsActive" -Method POST -ContentType "application/json" -Body $body
```

Ожидаемый результат:
```json
{
  "user": {
    "user_id": "u2",
    "username": "Dmitry",
    "team_name": "backend",
    "is_active": false
  }
}
```

HTTP статус: 200 OK

Пояснение: пользователь u2 помечен как неактивный. Это не удаляет его из существующих PR, где он уже назначен ревьювером, но исключает его из автоматического назначения на новые PR. Такой механизм полезен когда участник в отпуске или временно недоступен.

### Шаг 7. Переназначение ревьювера

Bash/Linux/Mac:
```bash
curl -X POST http://localhost:8080/pullRequest/reassign \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001",
    "old_user_id": "u2"
  }'
```

Windows PowerShell:
```powershell
$body = @'
{
  "pull_request_id": "pr-1001",
  "old_user_id": "u2"
}
'@
Invoke-RestMethod -Uri "http://localhost:8080/pullRequest/reassign" -Method POST -ContentType "application/json" -Body $body
```

Ожидаемый результат:
```json
{
  "pr": {
    "pull_request_id": "pr-1001",
    "pull_request_name": "Add search feature",
    "author_id": "u1",
    "status": "OPEN",
    "assigned_reviewers": ["u3", "u6"],
    "createdAt": "2025-11-14T10:30:00Z",
    "mergedAt": null
  },
  "replaced_by": "u6"
}
```

HTTP статус: 200 OK

Пояснение: ревьювер u2 заменен на u6. Алгоритм переназначения:
1. Проверяется что u2 действительно назначен ревьювером на pr-1001
2. Определяется команда заменяемого ревьювера (u2 из команды backend)
3. Формируется список кандидатов из команды backend, исключая:
   - Заменяемого ревьювера (u2)
   - Автора PR (u1)
   - Текущих ревьюверов (u3)
4. Из доступных кандидатов (u6) случайно выбирается один
5. u2 удаляется из списка ревьюверов, вместо него добавляется u6

Важно: новый ревьювер выбирается из команды заменяемого участника, а не из команды автора PR. Это позволяет сохранить распределение нагрузки внутри команды.

### Шаг 8. Слияние Pull Request

Bash/Linux/Mac:
```bash
curl -X POST http://localhost:8080/pullRequest/merge \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001"
  }'
```

Windows PowerShell:
```powershell
$body = @'
{
  "pull_request_id": "pr-1001"
}
'@
Invoke-RestMethod -Uri "http://localhost:8080/pullRequest/merge" -Method POST -ContentType "application/json" -Body $body
```

Ожидаемый результат:
```json
{
  "pr": {
    "pull_request_id": "pr-1001",
    "pull_request_name": "Add search feature",
    "author_id": "u1",
    "status": "MERGED",
    "assigned_reviewers": ["u3", "u6"],
    "createdAt": "2025-11-14T10:30:00Z",
    "mergedAt": "2025-11-14T10:45:00Z"
  }
}
```

HTTP статус: 200 OK

Пояснение: PR переведен в статус MERGED с фиксацией времени слияния. После этого список ревьюверов становится неизменяемым. Операция идемпотентна - повторный вызов вернет тот же результат без ошибки.

### Шаг 9. Попытка переназначить ревьювера после слияния

Bash/Linux/Mac:
```bash
curl -X POST http://localhost:8080/pullRequest/reassign \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001",
    "old_user_id": "u3"
  }'
```

Windows PowerShell:
```powershell
$body = @'
{
  "pull_request_id": "pr-1001",
  "old_user_id": "u3"
}
'@
Invoke-RestMethod -Uri "http://localhost:8080/pullRequest/reassign" -Method POST -ContentType "application/json" -Body $body
```

Ожидаемый результат:
```json
{
  "error": {
    "code": "PR_MERGED",
    "message": "PR_MERGED: cannot reassign on merged PR"
  }
}
```

HTTP статус: 409 Conflict

Пояснение: после слияния PR изменение списка ревьюверов запрещено. Это бизнес-требование обеспечивает историческую точность данных о том, кто именно ревьювил конкретный PR. Попытка изменить список после merge всегда возвращает ошибку PR_MERGED.

### Шаг 10. Повторное слияние того же PR (идемпотентность)

Bash/Linux/Mac:
```bash
curl -X POST http://localhost:8080/pullRequest/merge \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001"
  }'
```

Windows PowerShell:
```powershell
$body = @'
{
  "pull_request_id": "pr-1001"
}
'@
Invoke-RestMethod -Uri "http://localhost:8080/pullRequest/merge" -Method POST -ContentType "application/json" -Body $body
```

Ожидаемый результат:
```json
{
  "pr": {
    "pull_request_id": "pr-1001",
    "pull_request_name": "Add search feature",
    "author_id": "u1",
    "status": "MERGED",
    "assigned_reviewers": ["u3", "u6"],
    "createdAt": "2025-11-14T10:30:00Z",
    "mergedAt": "2025-11-14T10:45:00Z"
  }
}
```

HTTP статус: 200 OK

Пояснение: операция merge идемпотентна. Повторный вызов для уже слитого PR не вызывает ошибку, а возвращает актуальное состояние PR с кодом 200. Время слияния mergedAt остается неизменным (первоначальное время). Это упрощает интеграцию с внешними системами, которым не нужно отслеживать текущий статус PR перед вызовом merge.

### Дополнительные сценарии ошибок

Попытка создать PR для несуществующего пользователя:
```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "NOT_FOUND: author not found"
  }
}
```
HTTP статус: 404 Not Found

Попытка переназначить ревьювера, который не назначен на PR:
```json
{
  "error": {
    "code": "NOT_ASSIGNED",
    "message": "NOT_ASSIGNED: reviewer is not assigned to this PR"
  }
}
```
HTTP статус: 409 Conflict

Попытка переназначить ревьювера, когда нет доступных кандидатов:
```json
{
  "error": {
    "code": "NO_CANDIDATE",
    "message": "NO_CANDIDATE: no active replacement candidate in team"
  }
}
```
HTTP статус: 409 Conflict

Пояснение: ошибка NO_CANDIDATE возникает когда в команде заменяемого ревьювера не осталось активных участников, которые могли бы стать новым ревьювером (за исключением автора PR, текущих ревьюверов и самого заменяемого).

## Структура проекта

Проект организован по принципам чистой архитектуры с четким разделением на слои:

```
проект/
├── cmd/
│   └── server/          - точка входа приложения
├── internal/
│   ├── config/          - загрузка конфигурации
│   ├── logger/          - система логирования
│   ├── models/          - модели данных
│   ├── database/        - работа с БД
│   ├── repository/      - слой доступа к данным
│   ├── service/         - бизнес-логика
│   └── handlers/        - HTTP обработчики
├── migrations/          - SQL миграции
└── docker-compose.yml   - оркестрация контейнеров
```

Взаимодействие между слоями:
Handlers -> Service -> Repository -> Database

Каждый слой зависит только от интерфейсов нижележащих слоев. Dependency Injection используется для всех компонентов. Контекст передается через все слои для возможности отмены операций.

## Используемые технологии

- Go 1.23 - язык программирования
- PostgreSQL 17 - система управления базами данных
- pgx/v5 - драйвер для работы с PostgreSQL
- gorilla/mux - HTTP роутер
- golang-migrate - инструмент для миграций
- Docker & Docker Compose - контейнеризация

## Логика работы с ревьюверами

Создание PR:
При создании нового PR система автоматически выбирает до 2 активных участников из команды автора. Автор PR исключается из списка кандидатов. Если в команде меньше 2 доступных участников, назначается доступное количество (может быть 0 или 1).

Переназначение ревьювера:
Заменяемый ревьювер удаляется из списка, вместо него назначается случайный активный участник из команды заменяемого ревьювера (не автора PR). Из кандидатов исключаются:
- Заменяемый ревьювер
- Автор PR
- Все текущие ревьюверы данного PR

Слияние PR:
После слияния PR изменение списка ревьюверов становится невозможным. Операция слияния идемпотентна - повторный вызов не вызывает ошибку и возвращает актуальное состояние PR.

Статус активности:
Пользователи со статусом is_active = false не участвуют в автоматическом назначении на ревью.

## Коды ошибок

TEAM_EXISTS - попытка создать команду с существующим именем
PR_EXISTS - попытка создать PR с существующим идентификатором
PR_MERGED - попытка изменить PR после слияния
NOT_ASSIGNED - указанный пользователь не назначен ревьювером на данный PR
NO_CANDIDATE - нет доступных кандидатов для переназначения
NOT_FOUND - запрашиваемый ресурс не найден

## Локальная разработка

Установка зависимостей:
```bash
go mod download
```

Запуск только PostgreSQL:
```bash
docker-compose up postgres -d
```

Применение миграций:

Bash/Linux/Mac:
```bash
make migrate-up
```

Windows PowerShell (или если make недоступен):
```powershell
migrate -path migrations -database "postgresql://user:password@localhost:5432/pr_reviewer?sslmode=disable" up
```

Запуск сервиса локально:

Bash/Linux/Mac:
```bash
make run
```

Windows PowerShell:
```powershell
go run cmd/server/main.go
```

Сборка бинарного файла:

Bash/Linux/Mac:
```bash
make build
```

Windows PowerShell:
```powershell
go build -o bin/server.exe cmd/server/main.go
```

Запуск тестов:
```bash
make test
# или напрямую
go test -v -race -coverprofile=coverage.out ./...
```

Очистка временных файлов и остановка контейнеров:

Bash/Linux/Mac:
```bash
make clean
docker-compose down -v
```

Windows PowerShell (если make недоступен):
```powershell
# очистка бинарных файлов и coverage
Remove-Item -Recurse -Force bin -ErrorAction SilentlyContinue
Remove-Item coverage.out, coverage.html -ErrorAction SilentlyContinue

# остановка контейнеров
docker-compose down -v
```

## Принятые решения

Дополнительный эндпоинт для обновления команд:
Спецификация OpenAPI определяет что POST /team/add должен возвращать ошибку TEAM_EXISTS при попытке создать существующую команду. Для практического использования добавлен дополнительный эндпоинт POST /team/update, который позволяет обновлять состав существующих команд. Это решение соблюдает спецификацию и одновременно обеспечивает необходимый функционал для управления командами.

Переназначение из команды заменяемого:
Согласно техническому заданию, новый ревьювер выбирается из команды того участника, которого заменяют, а не из команды автора PR. Это позволяет сохранить распределение ревью внутри команды заменяемого.

Идемпотентность операции слияния:
Если PR уже находится в статусе MERGED, повторный вызов операции merge возвращает текущее состояние с кодом 200 вместо ошибки. Это упрощает интеграцию с внешними системами.

Исключения при переназначении:
Для предотвращения назначения одного человека несколько раз на один PR, система исключает из кандидатов:
- Заменяемого ревьювера
- Автора PR
- Всех текущих ревьюверов

Порядок полей в JSON:
Используется стандартная сериализация Go. Порядок полей может отличаться от примеров в спецификации OpenAPI, но структура данных полностью соответствует.

Генерация случайных значений:
Для выбора случайных ревьюверов используется пакет math/rand с перемешиванием списка кандидатов. Для повышения энтропии в production окружении рекомендуется использовать crypto/rand.

Эндпоинт статистики:
Добавлен дополнительный эндпоинт GET /stats для получения аналитики по пользователям. Для каждого пользователя показывается количество созданных PR, общее количество назначенных ревью и количество активных (открытых) ревью. Статистика упорядочена по количеству назначенных ревью в порядке убывания, что помогает увидеть наиболее загруженных участников команды. Это один из дополнительных заданий из технического задания.

## Переменные окружения

Все параметры настраиваются через переменные окружения:

PORT (по умолчанию 8080) - порт HTTP сервера
DATABASE_URL (обязательный) - строка подключения к PostgreSQL
DB_MAX_CONNS (по умолчанию 25) - максимум соединений в пуле
DB_MIN_CONNS (по умолчанию 5) - минимум соединений в пуле
LOG_LEVEL (по умолчанию info) - уровень логирования (debug, info, warn, error)

Пример конфигурации находится в файле .env.example

## Настройки производительности

Connection Pooling:
- Диапазон соединений с PostgreSQL: 5-25
- Период проверки соединений: 1 минута
- Максимальное время жизни соединения: 1 час
- Максимальное время простоя соединения: 30 минут

Индексы базы данных:
- Индекс на users.team_name для поиска по команде
- Индекс на users.is_active для фильтрации активных
- Индекс на pull_requests.status для фильтрации по статусу
- Индекс на pull_requests.author_id для поиска по автору
- Индекс на pull_request_reviewers.user_id для поиска PR по ревьюверу

Таймауты HTTP:
- Read Timeout: 15 секунд
- Write Timeout: 15 секунд
- Idle Timeout: 60 секунд
- Graceful Shutdown: 30 секунд

Транзакции:
- Создание команды с участниками выполняется в одной транзакции
- Создание PR с назначением ревьюверов в одной транзакции
- Переназначение ревьювера в одной транзакции

## Проверка работоспособности

Для проверки состояния сервиса используйте health check endpoint.

Bash/Linux/Mac:
```bash
curl http://localhost:8080/health
```

Windows PowerShell:
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/health" -Method GET
```

Ожидаемый(примерный) результат при успешной проверке:
```json
{
  "status": "healthy",
  "database": {
    "status": "healthy",
    "response_time": "7.038695ms"
  },
  "service": "pr-reviewer-service",
  "version": "1.0.0",
  "uptime": "1h4m45.721163252s"
}
```

HTTP статус: 200 OK

Пояснение: endpoint возвращает детальную информацию о состоянии сервиса, включая статус подключения к базе данных, время отклика БД, версию сервиса и время работы с момента запуска. Таймаут проверки составляет 2 секунды.

Если база данных недоступна:
```json
{
  "error": {
    "code": "UNHEALTHY",
    "message": "Database connection failed"
  }
}
```

HTTP статус: 503 Service Unavailable

## Работа с базой данных

Подключение к PostgreSQL в Docker контейнере:

```bash
docker exec -it pr_reviewer_db psql -U user -d pr_reviewer
```

Просмотр всех таблиц:

```sql
\dt
```

Просмотр структуры конкретной таблицы:

```sql
\d users
\d teams
\d pull_requests
\d pull_request_reviewers
```

Просмотр данных в таблице:

```sql
SELECT * FROM users;
SELECT * FROM teams;
SELECT * FROM pull_requests;
SELECT * FROM pull_request_reviewers;
```

Просмотр PR с ревьюверами:

```sql
SELECT
  pr.pull_request_id,
  pr.pull_request_name,
  pr.status,
  u.username as author,
  array_agg(r.username) as reviewers
FROM pull_requests pr
JOIN users u ON pr.author_id = u.user_id
LEFT JOIN pull_request_reviewers prr ON pr.pull_request_id = prr.pull_request_id
LEFT JOIN users r ON prr.user_id = r.user_id
GROUP BY pr.pull_request_id, pr.pull_request_name, pr.status, u.username;
```

Очистка всех таблиц:

```sql
TRUNCATE TABLE pull_request_reviewers, pull_requests, users, teams CASCADE;
```

Выход из psql:

```sql
\q
```

## Информация о проекте

Тестовое задание для Авито (осенняя волна 2025)

.env.example загрузил специально(не случайно)

Надеюсь на апрув))
