# Telegram бот для просмотра расписания занятий НКЛПиС

## Функционал

- 📅 Просмотр расписания
- 👥 Управление пользователями бота (админ)
- 📊 Статистика использования

## Требования

- Go 1.23 или выше
- Task (Task Runner)

## Установка

1. Клонируйте репозиторий:
```bash
git clone https://github.com/christmas-fire/nklpis_schedule_bot.git
cd nklpis_schedule_bot
```

2. Создайте файл `.env` в корневой директории:
```env
BOT_TOKEN=ваш_токен_бота
ADMIN=telegram_id_админа
```

3. Установите зависимости:
```bash
go mod download
```

## Запуск

### Сборка
```bash
task build
```

### Запуск
```bash
task run
```

### Просмотр логов
```bash
task logs
```

### Очистка
```bash
task clean
```
