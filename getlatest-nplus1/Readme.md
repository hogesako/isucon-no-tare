# なにがしたかった？
最新行だけを取得するようなN+1は何をするのがよさそうなのか知りたかった

# テーブル
isucon11予選からisuとisu_conditionだけ利用。

# 検証
```
echo "GET http://localhost:3000/api/isu" | vegeta attack -rate=100 -duration=10s | tee results.bin |vegeta report
```

## N+1 インデックスなし
### 結果
ブラウザ単発で300msくらい。スループットは5RPSくらい

### 内部ループのexplain analyze
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

### 内部ループのexplain analyze
実績値は3msくらいインデックスだけで取れている
```
 -> Limit: 1 row(s)  (cost=220.75 rows=1) (actual time=3.001..3.003 rows=1 loops=1)
    -> Index lookup on isu_condition using jia_isu_uuid_timestamp_idx (jia_isu_uuid='6892a276-e299-b319-1876-6d7bbaa1e176'), with index condition: (isu_condition.jia_isu_uuid = '6892a276-e299-b319-1876-6d7bbaa1e176')  (cost=220.75 rows=1000) (actual time=2.985..2.985 rows=1 loops=1)
```

```
```
## window関数

### SQL
```
SELECT
    i.id AS id,
    i.jia_isu_uuid AS jia_isu_uuid,
    i.name AS NAME,
    i.character AS `character`,
    ic.timestamp AS TIMESTAMP,
    ic.is_sitting AS is_sitting,
    ic.condition AS `condition`,
    ic.message AS message
FROM
    isu i
LEFT JOIN(
    SELECT *
    FROM
        (
        SELECT
            *,
            ROW_NUMBER() OVER(
            PARTITION BY jia_isu_uuid
        ORDER BY
            TIMESTAMP
        DESC
        ) AS latest_rank
    FROM
        isu_condition) isurank
        WHERE
            latest_rank = 1
    ) ic
ON
    i.jia_isu_uuid = ic.jia_isu_uuid
WHERE
    i.jia_user_id = 1
ORDER BY
    i.id
DESC;
```

### explain analyze
単発にはできているけど重い・・・
```
| -> Nested loop left join  (cost=26.12 rows=0) (actual time=281.296..281.547 rows=10 loops=1)
    -> Filter: (i.jia_user_id = 1)  (cost=1.25 rows=1) (actual time=0.630..0.709 rows=10 loops=1)
        -> Index scan on i using PRIMARY (reverse)  (cost=1.25 rows=10) (actual time=0.460..0.509 rows=10 loops=1)
    -> Index lookup on isurank using <auto_key0> (jia_isu_uuid=i.jia_isu_uuid, latest_rank=1)  (actual time=0.011..0.012 rows=1 loops=10)
        -> Materialize  (cost=0.00..0.00 rows=0) (actual time=28.053..28.056 rows=1 loops=10)
            -> Window aggregate: row_number() OVER (PARTITION BY isu_condition.jia_isu_uuid ORDER BY isu_condition.`timestamp` desc )   (actual time=102.402..167.483 rows=10000 loops=1)
                -> Sort: isu_condition.jia_isu_uuid, isu_condition.`timestamp` DESC  (cost=1034.85 rows=9946) (actual time=102.270..114.788 rows=10000 loops=1)
                    -> Table scan on isu_condition  (cost=1034.85 rows=9946) (actual time=0.671..42.693 rows=10000 loops=1)
```

## group by and max句でのサブクエリ
### SQL
```
select
    i.id AS id,
    i.jia_isu_uuid AS jia_isu_uuid,
    i.name AS NAME,
    i.character AS `character`,
    con.timestamp AS TIMESTAMP,
    con.is_sitting AS is_sitting,
    con.condition AS `condition`,
    con.message AS message
from isu i
LEFT JOIN
(select * from isu_condition con inner JOIN
(select MAX(timestamp) as max_timestamp,jia_isu_uuid as uuid from isu_condition latest group by jia_isu_uuid) latest_con
on con.jia_isu_uuid = latest_con.uuid and con.timestamp = latest_con.max_timestamp
) con
on i.jia_isu_uuid = con.jia_isu_uuid;
```

### explain analyze(indexなし)
384msでwindow関数のほうが早い
```
| -> Nested loop left join  (cost=259.85 rows=0) (actual time=88.533..384.893 rows=10 loops=1)
    -> Table scan on i  (cost=1.25 rows=10) (actual time=0.202..0.233 rows=10 loops=1)
    -> Nested loop inner join  (cost=413.16 rows=0) (actual time=22.238..38.457 rows=1 loops=10)
        -> Index lookup on latest_con using <auto_key1> (uuid=i.jia_isu_uuid)  (actual time=0.005..0.008 rows=1 loops=10)
            -> Materialize  (cost=0.00..0.00 rows=0) (actual time=8.812..8.817 rows=1 loops=10)
                -> Table scan on <temporary>  (actual time=0.014..0.022 rows=10 loops=1)
                    -> Aggregate using temporary table  (actual time=87.819..87.837 rows=10 loops=1)
                        -> Table scan on latest  (cost=1034.85 rows=9946) (actual time=0.395..32.577 rows=10000 loops=1)
        -> Filter: ((con.`timestamp` = latest_con.max_timestamp) and (con.jia_isu_uuid = i.jia_isu_uuid))  (cost=1.36 rows=9946) (actual time=13.421..29.634 rows=1 loops=10)
            -> Table scan on con  (cost=1.36 rows=9946) (actual time=0.084..22.265 rows=10000 loops=10)
```

### indexあり
group byとmax計算にインデックスが使用されるのでとても早い。
timestampのインデックスははASC,DESCどちらでも使用された。
```
| -> Nested loop left join  (cost=17.25 rows=110) (actual time=3.158..5.752 rows=10 loops=1)
    -> Table scan on i  (cost=1.25 rows=10) (actual time=0.458..0.575 rows=10 loops=1)
    -> Nested loop inner join  (cost=13.80 rows=11) (actual time=0.460..0.498 rows=1 loops=10)
        -> Index lookup on latest_con using <auto_key1> (uuid=i.jia_isu_uuid)  (actual time=0.018..0.024 rows=1 loops=10)
            -> Materialize  (cost=7.70..7.70 rows=11) (actual time=0.243..0.252 rows=1 loops=10)
                -> Index range scan on latest using index_for_group_by(jia_isu_uuid_timestamp_idx)  (cost=6.60 rows=11) (actual time=0.179..1.611 rows=10 loops=1)
        -> Index lookup on con using jia_isu_uuid_timestamp_idx (jia_isu_uuid=i.jia_isu_uuid, timestamp=latest_con.max_timestamp)  (cost=0.26 rows=1) (actual time=0.203..0.225 rows=1 loops=10)
```

## LATERAL
### SQL
```
select * from isu i
left join lateral (select * from isu_condition con where i.jia_isu_uuid = con.jia_isu_uuid order by timestamp desc limit 1) latest_con
on i.jia_isu_uuid = latest_con.jia_isu_uuid;
```

### indexなし
```
| -> Nested loop left join  (cost=1042.30 rows=10) (actual time=78.185..391.444 rows=10 loops=1)
    -> Invalidate materialized tables (row from i)  (cost=1.25 rows=10) (actual time=0.293..0.345 rows=10 loops=1)
        -> Table scan on i  (cost=1.25 rows=10) (actual time=0.285..0.326 rows=10 loops=1)
    -> Index lookup on latest_con using <auto_key0> (jia_isu_uuid=i.jia_isu_uuid)  (actual time=0.009..0.011 rows=1 loops=10)
        -> Materialize (invalidate on row from i)  (cost=139.81..139.81 rows=1) (actual time=39.102..39.104 rows=1 loops=10)
            -> Limit: 1 row(s)  (cost=139.71 rows=1) (actual time=39.063..39.064 rows=1 loops=10)
                -> Sort: con.`timestamp` DESC, limit input to 1 row(s) per chunk  (cost=139.71 rows=9946) (actual time=39.060..39.060 rows=1 loops=10)
                    -> Filter: (i.jia_isu_uuid = con.jia_isu_uuid)  (cost=139.71 rows=9946) (actual time=14.312..37.466 rows=1000 loops=10)
                        -> Table scan on con  (cost=139.71 rows=9946) (actual time=0.113..24.901 rows=10000 loops=10)
 |
```

### indexあり
124ms
```
| -> Nested loop left join  (cost=227.66 rows=10) (actual time=27.747..124.656 rows=10 loops=1)
    -> Invalidate materialized tables (row from i)  (cost=1.25 rows=10) (actual time=0.376..0.464 rows=10 loops=1)
        -> Table scan on i  (cost=1.25 rows=10) (actual time=0.316..0.375 rows=10 loops=1)
    -> Index lookup on latest_con using <auto_key0> (jia_isu_uuid=i.jia_isu_uuid)  (actual time=0.009..0.011 rows=1 loops=10)
        -> Materialize (invalidate on row from i)  (cost=220.31..220.31 rows=1) (actual time=12.390..12.393 rows=1 loops=10)
            -> Limit: 1 row(s)  (cost=220.21 rows=1) (actual time=12.347..12.348 rows=1 loops=10)
                -> Sort: con.`timestamp` DESC, limit input to 1 row(s) per chunk  (cost=220.21 rows=995) (actual time=12.342..12.342 rows=1 loops=10)
                    -> Index lookup on con using jia_isu_uuid_timestamp_idx (jia_isu_uuid=i.jia_isu_uuid)  (actual time=0.113..10.392 rows=1000 loops=10)
```

## 相関サブクエリ
### SQL
```
select * from isu i
left join
(select * from isu_condition con where con.timestamp = (select timestamp from isu_condition latest where latest.jia_isu_uuid = con.jia_isu_uuid order by timestamp desc limit 1)) cons
on cons.jia_isu_uuid = i.jia_isu_uuid;
```
### explain analyze
糞重い
```
| -> Nested loop left join  (cost=2203.35 rows=9946) (actual time=18.494..44468.229 rows=10 loops=1)
    -> Table scan on i  (cost=1.25 rows=10) (actual time=0.241..0.320 rows=10 loops=1)
    -> Filter: (con.`timestamp` = (select #3))  (cost=130.70 rows=995) (actual time=6.179..4446.771 rows=1 loops=10)
        -> Index lookup on con using jia_isu_uuid_timestamp_idx (jia_isu_uuid=i.jia_isu_uuid)  (cost=130.70 rows=995) (actual time=0.549..10.217 rows=1000 loops=10)
        -> Select #3 (subquery in condition; dependent)
            -> Limit: 1 row(s)  (cost=118.78 rows=1) (actual time=4.429..4.430 rows=1 loops=10000)
                -> Sort: latest.`timestamp` DESC, limit input to 1 row(s) per chunk  (cost=118.78 rows=995) (actual time=4.427..4.427 rows=1 loops=10000)
                    -> Index lookup on latest using jia_isu_uuid_timestamp_idx (jia_isu_uuid=con.jia_isu_uuid)  (actual time=0.033..3.348 rows=1000 loops=10000)
```

# 結論
WINDOW関数やLATERALではisu_conditionの集計時にindexが使われていない（新しい機能のため？）
そのため、現在ではjia_isu_uuid,timestampにindex貼りつつ古き良きサブクエリが早い