ALTER TABLE items ADD CONSTRAINT items_price_check CHECK (price >= 0);
ALTER TABLE items ADD CONSTRAINT items_supplier_check CHECK (supplier > 0);
ALTER TABLE items ADD CONSTRAINT items_currency_check CHECK (currency > 0);