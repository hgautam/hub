import { JSONSchema } from '@apidevtools/json-schema-ref-parser';
import { isUndefined } from 'lodash';
import React, { useEffect, useState } from 'react';

import compoundJSONSchemaYAML from '../../../utils/compoundJSONSchemaYAML';
import BlockCodeButtons from '../../common/BlockCodeButtons';
import styles from './Schema.module.css';
import SchemaLine from './SchemaLine';
import SchemaValuesSearch from './SchemaValuesSearch';

interface Props {
  schema: JSONSchema;
  normalizedName: string;
  definitions?: any;
}

const Schema = (props: Props) => {
  const [activePath, setActivePath] = useState<string | undefined>();
  const [valuesYAML, setValuesYAML] = useState<string | null>(null);
  const [availablePaths, setAvailablePaths] = useState<string[] | null>(null);

  useEffect(() => {
    const { yamlContent, paths } = compoundJSONSchemaYAML(props.schema);
    setValuesYAML(yamlContent);
    setAvailablePaths(paths);
  }, [props.schema]);

  const onSearch = (selectedPath?: string) => {
    setActivePath(selectedPath);
  };

  return (
    <>
      {availablePaths && <SchemaValuesSearch paths={availablePaths} activePath={activePath} onSearch={onSearch} />}
      <div className="row">
        <div className="col-7 pt-3 bg-dark position-relative">
          {props.schema.title && (
            <div className={`text-muted text-truncate text-monospace ${styles.comment}`}># {props.schema.title}</div>
          )}
          {valuesYAML && (
            <BlockCodeButtons
              className={styles.btns}
              content={valuesYAML}
              filename={`values-${props.normalizedName}.yaml`}
            />
          )}
        </div>
      </div>
      {!isUndefined(props.schema.properties) && (
        <>
          {Object.keys(props.schema.properties).map((propName: string) => {
            const value = props.schema.properties![propName] as JSONSchema;
            if (isUndefined(value)) return null;
            const isRequired = props.schema.required ? props.schema.required.includes(propName) : false;

            return (
              <React.Fragment key={propName}>
                <SchemaLine
                  value={value}
                  name={propName}
                  level={0}
                  isRequired={isRequired}
                  className="pt-4"
                  activePath={activePath}
                  setActivePath={setActivePath}
                />
              </React.Fragment>
            );
          })}
        </>
      )}
    </>
  );
};

export default Schema;
