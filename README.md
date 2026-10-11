## Publish Process

### Add tag

```
git tag v0.0.1
git push origin v0.0.1

Total 0 (delta 0), reused 0 (delta 0), pack-reused 0 (from 0)
To github.com:acernus18/pact.git
 * [new tag]         v0.0.1 -> v0.0.1
```

### Notify Go to cache it

```
git config --global http.proxy http://127.0.0.1:7890
git config --global https.proxy http://127.0.0.1:7890

go list -m github.com/acernus18/pact@v0.0.1

git config --global --unset http.proxy
git config --global --unset https.proxy
git config --global --list
```