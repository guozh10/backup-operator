kubectl create secret generic mysql-credentials \
  --from-literal=DATABASEUSER=root \
  --from-literal=DATABASEPASSWORD=password123 \
  --namespace=backup-system


# PostgreSQL 凭证
#kubectl create secret generic postgres-credentials \
#  --from-literal=username=postgres \
#  --from-literal=password=AnotherSecurePassword \
#  --namespace=staging



# 生成随机加密密钥
openssl rand -base64 32 > encryption-key.txt

kubectl create secret generic backup-encryption-key \
  --from-file=key=encryption-key.txt \
  --namespace=backup-system


