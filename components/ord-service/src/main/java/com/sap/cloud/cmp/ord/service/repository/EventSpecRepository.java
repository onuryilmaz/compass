package com.sap.cloud.cmp.ord.service.repository;

import com.sap.cloud.cmp.ord.service.storage.model.APISpecificationEntity;
import com.sap.cloud.cmp.ord.service.storage.model.EventSpecificationEntity;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;

import java.util.UUID;

@Repository
public interface EventSpecRepository extends JpaRepository<EventSpecificationEntity, UUID> {
}
