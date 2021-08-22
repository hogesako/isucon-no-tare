# なにがしたかった？
最新行だけを取得するようなN+1は何をするのがよさそうなのか知りたかった

# テーブル
isucon11予選からisuとisu_conditionだけ利用。

# 検証

## N+1 インデックスなし
### 結果
ブラウザ単発で300msくらい。スループットは5RPSくらい

### 内部ループの実行計画
74msぐらいかかっている。フルスキャン部分とソート部分が重い
```
explain analyze SELECT * FROM `isu_condition` WHERE `jia_isu_uuid` = '6892a276-e299-b319-1876-6d7bbaa1e176' ORDER BY `timestamp` DESC LIMIT 1
| -> Limit: 1 row(s)  (cost=1034.85 rows=1) (actual time=74.241..74.242 rows=1 loops=1)
    -> Sort: isu_condition.`timestamp` DESC, limit input to 1 row(s) per chunk  (cost=1034.85 rows=9946) (actual time=74.163..74.163 rows=1 loops=1)
        -> Filter: (isu_condition.jia_isu_uuid = '6892a276-e299-b319-1876-6d7bbaa1e176')  (cost=1034.85 rows=9946) (actual time=0.740..69.887 rows=1000 loops=1)
            -> Table scan on isu_condition  (cost=1034.85 rows=9946) (actual time=0.582..46.746 rows=10000 loops=1)
```



## N+1 with 最適なインデックス
### 結果
ブラウザ単発で180msくらい。スループットは10RPSくらい

### 内部ループの実行計画
実績値は3msくらいインデックスだけで取れている
```
 -> Limit: 1 row(s)  (cost=220.75 rows=1) (actual time=3.001..3.003 rows=1 loops=1)
    -> Index lookup on isu_condition using jia_isu_uuid_timestamp_idx (jia_isu_uuid='6892a276-e299-b319-1876-6d7bbaa1e176'), with index condition: (isu_condition.jia_isu_uuid = '6892a276-e299-b319-1876-6d7bbaa1e176')  (cost=220.75 rows=1000) (actual time=2.985..2.985 rows=1 loops=1)
```

```
```
## window関数

## group by and max句でのサブクエリ

## latestだけ持つテーブルを作る

## latestだけインメモリに持つ

## latestだけredisに持つ


# 結論