#!/bin/bash
echo "=== Полная зачистка LT-Plex ==="
sudo systemctl stop ltplex.service 2>/dev/null
sudo systemctl disable ltplex.service 2>/dev/null
sudo rm -f /etc/systemd/system/ltplex.service
sudo systemctl daemon-reload

sudo rm -rf /opt/ltplex

# Автоматически удаляем строку с ltplex.com из /etc/hosts
sudo sed -i '/ltplex.com/d' /etc/hosts

echo "Система полностью очищена!"